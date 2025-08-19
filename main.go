package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	db       *sql.DB
	router   *mux.Router
	hub      *Hub
	upgrader websocket.Upgrader
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func NewServer() (*Server, error) {
	db, err := initDB()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	hub := &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	s := &Server{
		db:     db,
		router: mux.NewRouter(),
		hub:    hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	s.setupRoutes()
	go s.hub.run()

	return s, nil
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Println("Client connected")

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Println("Client disconnected")
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (s *Server) setupRoutes() {
	// Static files
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	
	// HTML pages
	s.router.HandleFunc("/", s.handleHome).Methods("GET")
	s.router.HandleFunc("/submit-points", s.handleSubmitPointsPage).Methods("GET")
	s.router.HandleFunc("/login", s.handleLoginPage).Methods("GET")
	s.router.HandleFunc("/admin", s.handleAdminPage).Methods("GET")
	s.router.HandleFunc("/ward-log", s.handleWardLogPage).Methods("GET")
	
	// API endpoints
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/points", s.handleSubmitPoints).Methods("POST")
	api.HandleFunc("/points/{id}/approve", s.handleApprovePoints).Methods("POST")
	api.HandleFunc("/points/{id}/reject", s.handleRejectPoints).Methods("POST")
	api.HandleFunc("/leaderboard", s.handleGetLeaderboard).Methods("GET")
	api.HandleFunc("/auth/status", s.handleAuthStatus).Methods("GET")
	api.HandleFunc("/login", s.handleLogin).Methods("POST")
	api.HandleFunc("/logout", s.handleLogout).Methods("POST")
	api.HandleFunc("/user", s.handleGetUser).Methods("GET")
	api.HandleFunc("/submissions", s.handleGetSubmissions).Methods("GET")
	api.HandleFunc("/ward/{id}/log", s.handleGetWardLog).Methods("GET")
	
	// WebSocket endpoint
	s.router.HandleFunc("/ws", s.handleWebSocket)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "leaderboard.html")
}

func (s *Server) handleSubmitPointsPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "submit-points.html")
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "login.html")
}

func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "admin.html")
}

func (s *Server) handleWardLogPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "ward-log.html")
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:  s.hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Server) broadcastUpdate(updateType string, data interface{}) {
	message := map[string]interface{}{
		"type": updateType,
		"data": data,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling broadcast data: %v", err)
		return
	}

	s.hub.broadcast <- jsonData
}

func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, server.router); err != nil {
		log.Fatal(err)
	}
}