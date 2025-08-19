package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) handleGetLeaderboard(w http.ResponseWriter, r *http.Request) {
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "verified-desc"
	}

	// Get leaderboard entries
	entries, err := s.getLeaderboardEntries(sortBy)
	if err != nil {
		http.Error(w, "Failed to get leaderboard", http.StatusInternalServerError)
		log.Printf("Error getting leaderboard: %v", err)
		return
	}

	// Get stats
	stats, err := s.getStats()
	if err != nil {
		log.Printf("Error getting stats: %v", err)
	}

	response := map[string]interface{}{
		"leaderboard": entries,
		"stats":       stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getLeaderboardEntries(sortBy string) ([]LeaderboardEntry, error) {
	query := `
		SELECT 
			w.id,
			w.name,
			w.points,
			w.pending_points,
			w.points + w.pending_points as total_points,
			ROUND(CAST(w.points AS FLOAT) / 1300 * 100, 1) as progress
		FROM wards w
	`

	switch sortBy {
	case "verified-asc":
		query += " ORDER BY w.points ASC"
	case "total-desc":
		query += " ORDER BY total_points DESC"
	case "total-asc":
		query += " ORDER BY total_points ASC"
	case "ward-asc":
		query += " ORDER BY w.name ASC"
	case "ward-desc":
		query += " ORDER BY w.name DESC"
	default: // verified-desc
		query += " ORDER BY w.points DESC"
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	rank := 1

	for rows.Next() {
		var entry LeaderboardEntry
		err := rows.Scan(
			&entry.WardID,
			&entry.WardName,
			&entry.Points,
			&entry.PendingPoints,
			&entry.TotalPoints,
			&entry.Progress,
		)
		if err != nil {
			return nil, err
		}

		entry.Rank = rank
		rank++

		// Get achievements for this ward
		achievements, err := s.getWardAchievements(entry.WardID)
		if err != nil {
			log.Printf("Error getting achievements for ward %d: %v", entry.WardID, err)
		}
		entry.Achievements = achievements

		// Calculate streak (simplified for now)
		entry.Streak = s.calculateStreak(entry.WardID)

		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *Server) getWardAchievements(wardID int) ([]string, error) {
	query := `SELECT icon || ' ' || title FROM achievements WHERE ward_id = ?`
	rows, err := s.db.Query(query, wardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var achievements []string
	for rows.Next() {
		var achievement string
		if err := rows.Scan(&achievement); err != nil {
			return nil, err
		}
		achievements = append(achievements, achievement)
	}

	return achievements, nil
}

func (s *Server) calculateStreak(wardID int) int {
	// Simplified streak calculation - counts consecutive days with activity
	var streak int
	query := `
		SELECT COUNT(DISTINCT DATE(created_at)) as streak
		FROM activity_logs
		WHERE ward_id = ? 
		AND created_at >= datetime('now', '-7 days')
	`
	s.db.QueryRow(query, wardID).Scan(&streak)
	return streak
}

func (s *Server) getStats() (*Stats, error) {
	stats := &Stats{}

	// Get leading ward
	err := s.db.QueryRow(`
		SELECT name FROM wards ORDER BY points DESC LIMIT 1
	`).Scan(&stats.LeadingWard)
	if err != nil {
		return stats, err
	}

	// Get total points
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(points), 0) FROM wards
	`).Scan(&stats.TotalPoints)
	if err != nil {
		return stats, err
	}

	// Calculate days active (from first submission)
	var firstSubmission time.Time
	err = s.db.QueryRow(`
		SELECT MIN(created_at) FROM point_submissions WHERE status = 'approved'
	`).Scan(&firstSubmission)
	if err == nil {
		stats.DaysActive = int(time.Since(firstSubmission).Hours() / 24)
	}

	// Count unique participants
	err = s.db.QueryRow(`
		SELECT COUNT(DISTINCT submitter_name) FROM point_submissions
	`).Scan(&stats.Participants)
	
	return stats, nil
}

func (s *Server) handleSubmitPoints(w http.ResponseWriter, r *http.Request) {
	var submission struct {
		WardID        int    `json:"ward_id"`
		SubmitterName string `json:"submitter_name"`
		Points        int    `json:"points"`
		Note          string `json:"note"`
	}

	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate input
	if submission.WardID == 0 || submission.SubmitterName == "" || submission.Points <= 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Insert submission
	result, err := s.db.Exec(`
		INSERT INTO point_submissions (ward_id, submitter_name, points, note)
		VALUES (?, ?, ?, ?)
	`, submission.WardID, submission.SubmitterName, submission.Points, submission.Note)

	if err != nil {
		http.Error(w, "Failed to submit points", http.StatusInternalServerError)
		log.Printf("Error submitting points: %v", err)
		return
	}

	submissionID, _ := result.LastInsertId()

	// Update pending points for the ward
	_, err = s.db.Exec(`
		UPDATE wards 
		SET pending_points = (
			SELECT COALESCE(SUM(points), 0) 
			FROM point_submissions 
			WHERE ward_id = ? AND status = 'pending'
		)
		WHERE id = ?
	`, submission.WardID, submission.WardID)

	if err != nil {
		log.Printf("Error updating pending points: %v", err)
	}

	// Log activity
	s.logActivity(submission.WardID, nil, "points_submitted", 
		fmt.Sprintf("%s submitted %d points", submission.SubmitterName, submission.Points), 
		submission.Points)

	// Broadcast update to all connected clients
	s.broadcastLeaderboardUpdate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      submissionID,
		"message": "Points submitted successfully! Waiting for approval.",
	})
}

func (s *Server) handleApprovePoints(w http.ResponseWriter, r *http.Request) {
	// Check authentication (simplified for now)
	userID := s.getUserIDFromSession(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	submissionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid submission ID", http.StatusBadRequest)
		return
	}

	// Get submission details
	var wardID, points int
	var submitterName string
	err = s.db.QueryRow(`
		SELECT ward_id, points, submitter_name 
		FROM point_submissions 
		WHERE id = ? AND status = 'pending'
	`, submissionID).Scan(&wardID, &points, &submitterName)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Submission not found or already processed", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Check if user can approve for this ward
	if !s.canApproveForWard(userID, wardID) {
		http.Error(w, "Not authorized to approve for this ward", http.StatusForbidden)
		return
	}

	// Approve the submission
	_, err = s.db.Exec(`
		UPDATE point_submissions 
		SET status = 'approved', approved_by = ?, approved_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, userID, submissionID)

	if err != nil {
		http.Error(w, "Failed to approve submission", http.StatusInternalServerError)
		return
	}

	// Update ward points
	_, err = s.db.Exec(`
		UPDATE wards 
		SET points = points + ?,
		    pending_points = pending_points - ?
		WHERE id = ?
	`, points, points, wardID)

	if err != nil {
		log.Printf("Error updating ward points: %v", err)
	}

	// Check for achievements
	s.checkAndAwardAchievements(wardID)

	// Log activity
	s.logActivity(wardID, &userID, "points_approved", 
		fmt.Sprintf("Approved %d points from %s", points, submitterName), points)

	// Broadcast update
	s.broadcastLeaderboardUpdate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Points approved successfully!",
	})
}

func (s *Server) handleRejectPoints(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserIDFromSession(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	submissionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid submission ID", http.StatusBadRequest)
		return
	}

	// Get submission details
	var wardID, points int
	err = s.db.QueryRow(`
		SELECT ward_id, points 
		FROM point_submissions 
		WHERE id = ? AND status = 'pending'
	`, submissionID).Scan(&wardID, &points)

	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	// Check authorization
	if !s.canApproveForWard(userID, wardID) {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	// Reject the submission
	_, err = s.db.Exec(`
		UPDATE point_submissions 
		SET status = 'rejected', approved_by = ?, approved_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, userID, submissionID)

	if err != nil {
		http.Error(w, "Failed to reject submission", http.StatusInternalServerError)
		return
	}

	// Update pending points
	_, err = s.db.Exec(`
		UPDATE wards 
		SET pending_points = pending_points - ?
		WHERE id = ?
	`, points, wardID)

	// Broadcast update
	s.broadcastLeaderboardUpdate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Points rejected",
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var user User
	var hashedPassword string
	err := s.db.QueryRow(`
		SELECT id, email, password, role, ward_id 
		FROM users 
		WHERE email = ?
	`, credentials.Email).Scan(&user.ID, &user.Email, &hashedPassword, &user.Role, &user.WardID)

	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(credentials.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Create session (simplified - in production use proper session management)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    fmt.Sprintf("%d", user.ID),
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400, // 24 hours
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserIDFromSession(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user User
	err := s.db.QueryRow(`
		SELECT id, email, role, ward_id 
		FROM users 
		WHERE id = ?
	`, userID).Scan(&user.ID, &user.Email, &user.Role, &user.WardID)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleGetWardLog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	wardID := vars["id"]

	// Get ward info
	var wardName string
	var totalPoints, pendingPoints int
	err := s.db.QueryRow(`
		SELECT name, points, pending_points 
		FROM wards 
		WHERE id = ?
	`, wardID).Scan(&wardName, &totalPoints, &pendingPoints)

	if err != nil {
		http.Error(w, "Ward not found", http.StatusNotFound)
		return
	}

	// Get all submissions for this ward
	query := `
		SELECT id, submitter_name, points, note, status, created_at
		FROM point_submissions
		WHERE ward_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, wardID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Error querying ward submissions: %v", err)
		return
	}
	defer rows.Close()

	var submissions []PointSubmission
	for rows.Next() {
		var sub PointSubmission
		err := rows.Scan(&sub.ID, &sub.SubmitterName, &sub.Points, 
			&sub.Note, &sub.Status, &sub.CreatedAt)
		if err != nil {
			log.Printf("Error scanning submission: %v", err)
			continue
		}
		submissions = append(submissions, sub)
	}

	response := map[string]interface{}{
		"ward_id":        wardID,
		"ward_name":      wardName,
		"total_points":   totalPoints,
		"pending_points": pendingPoints,
		"submissions":    submissions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetSubmissions(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserIDFromSession(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}

	// Check user role and ward
	var role string
	var userWardID sql.NullInt64
	err := s.db.QueryRow(`
		SELECT role, ward_id FROM users WHERE id = ?
	`, userID).Scan(&role, &userWardID)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var query string
	var args []interface{}

	if role == "admin" {
		// Admin can see all submissions
		query = `
			SELECT ps.id, ps.ward_id, w.name, ps.submitter_name, ps.points, 
			       ps.note, ps.status, ps.created_at
			FROM point_submissions ps
			JOIN wards w ON ps.ward_id = w.id
			WHERE ps.status = ?
			ORDER BY ps.created_at DESC
			LIMIT 50
		`
		args = []interface{}{status}
	} else if role == "ward_approver" && userWardID.Valid {
		// Ward approver can only see their ward's submissions
		query = `
			SELECT ps.id, ps.ward_id, w.name, ps.submitter_name, ps.points, 
			       ps.note, ps.status, ps.created_at
			FROM point_submissions ps
			JOIN wards w ON ps.ward_id = w.id
			WHERE ps.status = ? AND ps.ward_id = ?
			ORDER BY ps.created_at DESC
			LIMIT 50
		`
		args = []interface{}{status, userWardID.Int64}
	} else {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Error querying submissions: %v", err)
		return
	}
	defer rows.Close()

	var submissions []PointSubmission
	for rows.Next() {
		var sub PointSubmission
		err := rows.Scan(&sub.ID, &sub.WardID, &sub.WardName, &sub.SubmitterName,
			&sub.Points, &sub.Note, &sub.Status, &sub.CreatedAt)
		if err != nil {
			log.Printf("Error scanning submission: %v", err)
			continue
		}
		submissions = append(submissions, sub)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(submissions)
}

// Helper functions

func (s *Server) getUserIDFromSession(r *http.Request) int {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0
	}

	userID, err := strconv.Atoi(cookie.Value)
	if err != nil {
		return 0
	}

	return userID
}

func (s *Server) canApproveForWard(userID, wardID int) bool {
	var role string
	var userWardID sql.NullInt64
	
	err := s.db.QueryRow(`
		SELECT role, ward_id FROM users WHERE id = ?
	`, userID).Scan(&role, &userWardID)

	if err != nil {
		return false
	}

	// Admins can approve for any ward
	if role == "admin" {
		return true
	}

	// Ward approvers can only approve for their ward
	if role == "ward_approver" && userWardID.Valid && int(userWardID.Int64) == wardID {
		return true
	}

	return false
}

func (s *Server) logActivity(wardID int, userID *int, action, details string, points int) {
	_, err := s.db.Exec(`
		INSERT INTO activity_logs (ward_id, user_id, action, details, points)
		VALUES (?, ?, ?, ?, ?)
	`, wardID, userID, action, details, points)

	if err != nil {
		log.Printf("Error logging activity: %v", err)
	}
}

func (s *Server) checkAndAwardAchievements(wardID int) {
	// Get current ward points
	var points int
	s.db.QueryRow("SELECT points FROM wards WHERE id = ?", wardID).Scan(&points)

	// Check various achievement conditions
	achievements := []struct {
		condition bool
		aType     string
		title     string
		icon      string
	}{
		{points >= 100, "first_100", "First 100 Points!", "💯"},
		{points >= 500, "first_500", "First to 500!", "⚡"},
		{points >= 1000, "first_1000", "Thousand Club!", "🎯"},
		{points >= 1300, "goal_reached", "Goal Achieved!", "🏆"},
	}

	for _, ach := range achievements {
		if ach.condition {
			_, err := s.db.Exec(`
				INSERT OR IGNORE INTO achievements (ward_id, type, title, icon)
				VALUES (?, ?, ?, ?)
			`, wardID, ach.aType, ach.title, ach.icon)

			if err == nil {
				// If this was a new achievement, broadcast it
				s.broadcastAchievement(wardID, ach.title)
			}
		}
	}
}

func (s *Server) broadcastLeaderboardUpdate() {
	entries, _ := s.getLeaderboardEntries("verified-desc")
	stats, _ := s.getStats()

	s.broadcastUpdate("leaderboard-update", map[string]interface{}{
		"leaderboard": entries,
		"stats":       stats,
	})
}

func (s *Server) broadcastAchievement(wardID int, achievement string) {
	var wardName string
	s.db.QueryRow("SELECT name FROM wards WHERE id = ?", wardID).Scan(&wardName)

	s.broadcastUpdate("achievement", map[string]interface{}{
		"ward":        wardName,
		"achievement": achievement,
		"milestone":   fmt.Sprintf("%s earned: %s", wardName, achievement),
	})
}