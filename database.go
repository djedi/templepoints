package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./templepoints.db?_foreign_keys=on")
	if err != nil {
		return nil, err
	}

	if err := createTables(db); err != nil {
		return nil, err
	}

	if err := seedData(db); err != nil {
		return nil, err
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS wards (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		points INTEGER DEFAULT 0,
		pending_points INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		role TEXT NOT NULL CHECK(role IN ('admin', 'ward_approver')),
		ward_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ward_id) REFERENCES wards(id)
	);

	CREATE TABLE IF NOT EXISTS point_submissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ward_id INTEGER NOT NULL,
		submitter_name TEXT NOT NULL,
		points INTEGER NOT NULL,
		note TEXT,
		status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'rejected')),
		approved_by INTEGER,
		approved_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ward_id) REFERENCES wards(id),
		FOREIGN KEY (approved_by) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS achievements (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ward_id INTEGER NOT NULL,
		type TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		icon TEXT,
		earned_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ward_id) REFERENCES wards(id),
		UNIQUE(ward_id, type)
	);

	CREATE TABLE IF NOT EXISTS activity_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ward_id INTEGER NOT NULL,
		user_id INTEGER,
		action TEXT NOT NULL,
		details TEXT,
		points INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (ward_id) REFERENCES wards(id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE INDEX IF NOT EXISTS idx_submissions_status ON point_submissions(status);
	CREATE INDEX IF NOT EXISTS idx_submissions_ward ON point_submissions(ward_id);
	CREATE INDEX IF NOT EXISTS idx_achievements_ward ON achievements(ward_id);
	CREATE INDEX IF NOT EXISTS idx_activity_ward ON activity_logs(ward_id);
	CREATE INDEX IF NOT EXISTS idx_activity_created ON activity_logs(created_at);
	`

	_, err := db.Exec(schema)
	return err
}

func seedData(db *sql.DB) error {
	// Check if wards already exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM wards").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Already seeded
	}

	// Seed wards
	wards := []string{
		"Fountain Green 1st Ward",
		"Fountain Green 2nd Ward",
		"Fountain Green 3rd Ward",
		"Moroni 1st Ward",
		"Moroni 2nd Ward",
		"Moroni 3rd Ward",
		"Sanpitch Ward",
	}

	for _, ward := range wards {
		_, err := db.Exec("INSERT INTO wards (name) VALUES (?)", ward)
		if err != nil {
			return fmt.Errorf("failed to insert ward %s: %w", ward, err)
		}
	}

	// Create default admin user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		"INSERT INTO users (email, password, role) VALUES (?, ?, ?)",
		"admin@templepoints.org",
		string(hashedPassword),
		"admin",
	)
	if err != nil {
		log.Printf("Admin user might already exist: %v", err)
	}

	// Add some sample data for demo
	sampleSubmissions := []struct {
		wardID int
		points int
		status string
	}{
		{4, 847, "approved"}, // Moroni 1st
		{2, 765, "approved"}, // FG 2nd
		{7, 692, "approved"}, // Sanpitch
		{6, 543, "approved"}, // Moroni 3rd
		{1, 489, "approved"}, // FG 1st
		{5, 412, "approved"}, // Moroni 2nd
		{3, 387, "approved"}, // FG 3rd
		// Pending points
		{4, 50, "pending"},
		{2, 55, "pending"},
		{6, 35, "pending"},
		{1, 25, "pending"},
		{3, 55, "pending"},
	}

	for _, sub := range sampleSubmissions {
		_, err := db.Exec(
			`INSERT INTO point_submissions (ward_id, submitter_name, points, note, status, approved_at) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			sub.wardID, "Demo User", sub.points, "Initial seed data", sub.status,
			func() *time.Time {
				if sub.status == "approved" {
					t := time.Now()
					return &t
				}
				return nil
			}(),
		)
		if err != nil {
			log.Printf("Error inserting sample submission: %v", err)
		}
	}

	// Update ward points based on approved submissions
	_, err = db.Exec(`
		UPDATE wards 
		SET points = (
			SELECT COALESCE(SUM(points), 0) 
			FROM point_submissions 
			WHERE ward_id = wards.id AND status = 'approved'
		),
		pending_points = (
			SELECT COALESCE(SUM(points), 0) 
			FROM point_submissions 
			WHERE ward_id = wards.id AND status = 'pending'
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to update ward points: %w", err)
	}

	// Add some achievements
	achievements := []struct {
		wardID int
		aType  string
		title  string
		icon   string
	}{
		{4, "first_500", "First to 500!", "⚡"},
		{4, "week_champion", "Week Champion", "🏆"},
		{4, "streak_3", "3 Day Streak!", "🔥"},
		{2, "rising_star", "Rising Star", "✨"},
		{2, "team_player", "Team Player", "🤝"},
		{7, "consistent", "Consistent", "📊"},
		{6, "getting_started", "Getting Started", "🌟"},
		{3, "momentum", "Building Momentum!", "🚀"},
	}

	for _, ach := range achievements {
		_, err := db.Exec(
			"INSERT OR IGNORE INTO achievements (ward_id, type, title, icon) VALUES (?, ?, ?, ?)",
			ach.wardID, ach.aType, ach.title, ach.icon,
		)
		if err != nil {
			log.Printf("Error inserting achievement: %v", err)
		}
	}

	log.Println("Database seeded successfully")
	return nil
}