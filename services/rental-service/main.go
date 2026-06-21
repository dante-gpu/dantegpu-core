package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RentalRequest struct {
	GPUInstanceID string  `json:"gpu_instance_id"`
	DurationHours float64 `json:"duration_hours"`
}

type Rental struct {
	ID             string                 `json:"id" db:"id"`
	UserID         string                 `json:"user_id" db:"user_id"`
	GPUInstanceID  string                 `json:"gpu_instance_id" db:"gpu_instance_id"`
	Status         string                 `json:"status" db:"status"`
	StartTime      *time.Time             `json:"start_time" db:"start_time"`
	EndTime        *time.Time             `json:"end_time" db:"end_time"`
	DurationHours  *float64               `json:"duration_hours" db:"duration_hours"`
	PricePerHour   float64                `json:"price_per_hour" db:"price_per_hour"`
	TotalCost      *float64               `json:"total_cost" db:"total_cost"`
	PaymentStatus  string                 `json:"payment_status" db:"payment_status"`
	ConnectionInfo map[string]interface{} `json:"connection_info" db:"connection_info"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

type RentalService struct {
	db *pgxpool.Pool
}

func NewRentalService(db *pgxpool.Pool) *RentalService {
	return &RentalService{db: db}
}

func (s *RentalService) createRental(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req RentalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if GPU is available
	var gpuStatus string
	var pricePerHour float64
	err := s.db.QueryRow(context.Background(),
		"SELECT status, price_per_hour FROM gpu_instances WHERE id = $1",
		req.GPUInstanceID).Scan(&gpuStatus, &pricePerHour)

	if err != nil {
		http.Error(w, "GPU not found", http.StatusNotFound)
		return
	}

	if gpuStatus != "available" {
		http.Error(w, "GPU not available", http.StatusConflict)
		return
	}

	// Check user balance
	var userBalance float64
	err = s.db.QueryRow(context.Background(),
		"SELECT balance FROM users WHERE id = $1", userID).Scan(&userBalance)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	estimatedCost := req.DurationHours * pricePerHour
	if userBalance < estimatedCost {
		http.Error(w, "Insufficient balance", http.StatusPaymentRequired)
		return
	}

	// Start transaction
	tx, err := s.db.Begin(context.Background())
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(context.Background())

	// Create rental
	var rental Rental
	err = tx.QueryRow(context.Background(),
		`INSERT INTO gpu_rentals (user_id, gpu_instance_id, status, price_per_hour, payment_status) 
		 VALUES ($1, $2, 'pending', $3, 'pending') 
		 RETURNING id, user_id, gpu_instance_id, status, start_time, end_time, duration_hours, 
		           price_per_hour, total_cost, payment_status, connection_info, created_at, updated_at`,
		userID, req.GPUInstanceID, pricePerHour).Scan(
		&rental.ID, &rental.UserID, &rental.GPUInstanceID, &rental.Status, &rental.StartTime,
		&rental.EndTime, &rental.DurationHours, &rental.PricePerHour, &rental.TotalCost,
		&rental.PaymentStatus, &rental.ConnectionInfo, &rental.CreatedAt, &rental.UpdatedAt)

	if err != nil {
		http.Error(w, "Failed to create rental", http.StatusInternalServerError)
		return
	}

	// Update GPU status to busy
	_, err = tx.Exec(context.Background(),
		"UPDATE gpu_instances SET status = 'busy' WHERE id = $1", req.GPUInstanceID)

	if err != nil {
		http.Error(w, "Failed to update GPU status", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err = tx.Commit(context.Background()); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rental)
}

func (s *RentalService) startRental(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	rentalID := vars["id"]

	// Start transaction
	tx, err := s.db.Begin(context.Background())
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(context.Background())

	// Update rental status and start time
	now := time.Now()
	_, err = tx.Exec(context.Background(),
		`UPDATE gpu_rentals 
		 SET status = 'running', start_time = $1, payment_status = 'paid' 
		 WHERE id = $2 AND user_id = $3 AND status = 'pending'`,
		now, rentalID, userID)

	if err != nil {
		http.Error(w, "Failed to start rental", http.StatusInternalServerError)
		return
	}

	// Generate connection info (mock)
	connectionInfo := map[string]interface{}{
		"ssh_host":     "gpu-instance.dantegpu.com",
		"ssh_port":     22,
		"ssh_username": "ubuntu",
		"ssh_password": "generated_password_123",
		"jupyter_url":  "https://jupyter.dantegpu.com/lab",
		"vnc_url":      "https://vnc.dantegpu.com",
	}

	connectionInfoJSON, _ := json.Marshal(connectionInfo)

	// Update connection info
	_, err = tx.Exec(context.Background(),
		"UPDATE gpu_rentals SET connection_info = $1 WHERE id = $2",
		connectionInfoJSON, rentalID)

	if err != nil {
		http.Error(w, "Failed to update connection info", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err = tx.Commit(context.Background()); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "started",
		"connection_info": connectionInfo,
	})
}

func (s *RentalService) stopRental(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	rentalID := vars["id"]

	// Start transaction
	tx, err := s.db.Begin(context.Background())
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(context.Background())

	// Get rental info
	var startTime time.Time
	var pricePerHour float64
	var gpuInstanceID string

	err = tx.QueryRow(context.Background(),
		`SELECT start_time, price_per_hour, gpu_instance_id 
		 FROM gpu_rentals 
		 WHERE id = $1 AND user_id = $2 AND status = 'running'`,
		rentalID, userID).Scan(&startTime, &pricePerHour, &gpuInstanceID)

	if err != nil {
		http.Error(w, "Rental not found or not running", http.StatusNotFound)
		return
	}

	// Calculate duration and cost
	now := time.Now()
	duration := now.Sub(startTime).Hours()
	totalCost := duration * pricePerHour

	// Update rental
	_, err = tx.Exec(context.Background(),
		`UPDATE gpu_rentals 
		 SET status = 'completed', end_time = $1, duration_hours = $2, total_cost = $3 
		 WHERE id = $4`,
		now, duration, totalCost, rentalID)

	if err != nil {
		http.Error(w, "Failed to stop rental", http.StatusInternalServerError)
		return
	}

	// Update GPU status back to available
	_, err = tx.Exec(context.Background(),
		"UPDATE gpu_instances SET status = 'available' WHERE id = $1", gpuInstanceID)

	if err != nil {
		http.Error(w, "Failed to update GPU status", http.StatusInternalServerError)
		return
	}

	// Deduct cost from user balance
	_, err = tx.Exec(context.Background(),
		"UPDATE users SET balance = balance - $1 WHERE id = $2", totalCost, userID)

	if err != nil {
		http.Error(w, "Failed to update user balance", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err = tx.Commit(context.Background()); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "stopped",
		"duration":   duration,
		"total_cost": totalCost,
		"stopped_at": now,
	})
}

func (s *RentalService) getUserRentals(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := s.db.Query(context.Background(),
		`SELECT id, user_id, gpu_instance_id, status, start_time, end_time, duration_hours, 
		        price_per_hour, total_cost, payment_status, connection_info, created_at, updated_at
		 FROM gpu_rentals 
		 WHERE user_id = $1 
		 ORDER BY created_at DESC`,
		userID)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var rentals []Rental
	for rows.Next() {
		var rental Rental
		var connectionInfoJSON []byte

		err := rows.Scan(
			&rental.ID, &rental.UserID, &rental.GPUInstanceID, &rental.Status,
			&rental.StartTime, &rental.EndTime, &rental.DurationHours,
			&rental.PricePerHour, &rental.TotalCost, &rental.PaymentStatus,
			&connectionInfoJSON, &rental.CreatedAt, &rental.UpdatedAt,
		)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}

		if connectionInfoJSON != nil {
			json.Unmarshal(connectionInfoJSON, &rental.ConnectionInfo)
		}

		rentals = append(rentals, rental)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rentals)
}

func (s *RentalService) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func main() {
	// Database connection
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgresql://dantegpu:dantegpu123@localhost:5432/dantegpu"
	}

	db, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	rentalService := NewRentalService(db)

	r := mux.NewRouter()

	// CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Routes
	r.HandleFunc("/health", rentalService.health).Methods("GET")
	r.HandleFunc("/rentals", rentalService.createRental).Methods("POST")
	r.HandleFunc("/rentals", rentalService.getUserRentals).Methods("GET")
	r.HandleFunc("/rentals/{id}/start", rentalService.startRental).Methods("POST")
	r.HandleFunc("/rentals/{id}/stop", rentalService.stopRental).Methods("POST")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8092"
	}

	fmt.Printf("Rental service starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
