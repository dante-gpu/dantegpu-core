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
	_ "github.com/mattn/go-sqlite3"
)

type GPU struct {
	ID           string  `json:"id"`
	Model        string  `json:"model"`
	VRAM         string  `json:"vram"`
	CUDACores    int     `json:"cuda_cores"`
	PricePerHour float64 `json:"price_per_hour"`
	ProviderID   string  `json:"provider_id"`
	ProviderName string  `json:"provider_name"`
	Location     string  `json:"location"`
	Status       string  `json:"status"`
	Utilization  int     `json:"utilization"`
	Temperature  int     `json:"temperature"`
}

type Rental struct {
	ID           string    `json:"id"`
	GPUID        string    `json:"gpu_id"`
	UserID       string    `json:"user_id"`
	EscrowAmount float64   `json:"escrow_amount"`
	HourlyRate   float64   `json:"hourly_rate"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
	EstimatedEnd time.Time `json:"estimated_end"`
}

type Job struct {
	ID          string    `json:"id"`
	RentalID    string    `json:"rental_id"`
	DockerImage string    `json:"docker_image"`
	Command     string    `json:"command"`
	Status      string    `json:"status"`
	Progress    int       `json:"progress"`
	CreatedAt   time.Time `json:"created_at"`
}

type Wallet struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Address   string    `json:"address"`
	Balance   float64   `json:"balance"`
	Available float64   `json:"available"`
	Locked    float64   `json:"locked"`
	Network   string    `json:"network"`
	CreatedAt time.Time `json:"created_at"`
}

type APIGateway struct {
	db *sql.DB
}

func NewAPIGateway(db *sql.DB) *APIGateway {
	return &APIGateway{db: db}
}

func (gw *APIGateway) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS gpus (
		id TEXT PRIMARY KEY,
		model TEXT NOT NULL,
		vram TEXT NOT NULL,
		cuda_cores INTEGER NOT NULL,
		price_per_hour REAL NOT NULL,
		provider_id TEXT NOT NULL,
		provider_name TEXT NOT NULL,
		location TEXT NOT NULL,
		status TEXT DEFAULT 'available',
		utilization INTEGER DEFAULT 0,
		temperature INTEGER DEFAULT 45,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS rentals (
		id TEXT PRIMARY KEY,
		gpu_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		escrow_amount REAL NOT NULL,
		hourly_rate REAL NOT NULL,
		status TEXT DEFAULT 'active',
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		estimated_end DATETIME,
		FOREIGN KEY (gpu_id) REFERENCES gpus(id)
	);

	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		rental_id TEXT NOT NULL,
		docker_image TEXT NOT NULL,
		command TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		progress INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (rental_id) REFERENCES rentals(id)
	);

	CREATE TABLE IF NOT EXISTS wallets (
		id TEXT PRIMARY KEY,
		user_id TEXT UNIQUE NOT NULL,
		address TEXT UNIQUE NOT NULL,
		balance REAL DEFAULT 0.0,
		available REAL DEFAULT 0.0,
		locked REAL DEFAULT 0.0,
		network TEXT DEFAULT 'solana-mainnet',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Insert sample GPUs
	INSERT OR IGNORE INTO gpus (id, model, vram, cuda_cores, price_per_hour, provider_id, provider_name, location, status, utilization, temperature)
	VALUES 
		('gpu_001', 'NVIDIA RTX 4090', '24GB', 16384, 2.5, 'provider_001', 'TechProvider Inc.', 'US-West', 'available', 0, 45),
		('gpu_002', 'NVIDIA A100', '40GB', 6912, 5.0, 'provider_002', 'CloudGPU Ltd.', 'EU-Central', 'available', 0, 42),
		('gpu_003', 'NVIDIA RTX 3090', '24GB', 10496, 1.8, 'provider_003', 'GPUFarm Co.', 'Asia-East', 'available', 0, 48),
		('gpu_004', 'NVIDIA H100', '80GB', 16896, 8.0, 'provider_004', 'AI Compute Inc.', 'US-East', 'available', 0, 40),
		('gpu_005', 'NVIDIA A40', '48GB', 10752, 3.5, 'provider_005', 'DataCenter Pro', 'EU-West', 'available', 0, 43);
	`

	_, err := gw.db.Exec(schema)
	return err
}

func (gw *APIGateway) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (gw *APIGateway) sendError(w http.ResponseWriter, statusCode int, message string) {
	gw.sendJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// GPU Handlers
func (gw *APIGateway) listGPUs(w http.ResponseWriter, r *http.Request) {
	rows, err := gw.db.Query(`
		SELECT id, model, vram, cuda_cores, price_per_hour, provider_id, provider_name, 
		       location, status, utilization, temperature 
		FROM gpus 
		WHERE status = 'available'
		ORDER BY price_per_hour ASC
	`)
	if err != nil {
		log.Printf("Database error: %v", err)
		gw.sendError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var gpus []GPU
	for rows.Next() {
		var gpu GPU
		err := rows.Scan(&gpu.ID, &gpu.Model, &gpu.VRAM, &gpu.CUDACores, &gpu.PricePerHour,
			&gpu.ProviderID, &gpu.ProviderName, &gpu.Location, &gpu.Status, &gpu.Utilization, &gpu.Temperature)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		gpus = append(gpus, gpu)
	}

	gw.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"gpus":    gpus,
		"total":   len(gpus),
	})
}

func (gw *APIGateway) getGPU(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gpuID := vars["id"]

	var gpu GPU
	err := gw.db.QueryRow(`
		SELECT id, model, vram, cuda_cores, price_per_hour, provider_id, provider_name,
		       location, status, utilization, temperature
		FROM gpus WHERE id = ?
	`, gpuID).Scan(&gpu.ID, &gpu.Model, &gpu.VRAM, &gpu.CUDACores, &gpu.PricePerHour,
		&gpu.ProviderID, &gpu.ProviderName, &gpu.Location, &gpu.Status, &gpu.Utilization, &gpu.Temperature)

	if err == sql.ErrNoRows {
		gw.sendError(w, http.StatusNotFound, "GPU not found")
		return
	}

	if err != nil {
		log.Printf("Database error: %v", err)
		gw.sendError(w, http.StatusInternalServerError, "Database error")
		return
	}

	gw.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"gpu":     gpu,
	})
}

// Wallet Handlers
func (gw *APIGateway) createWallet(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		gw.sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Check if wallet already exists
	var exists int
	err := gw.db.QueryRow("SELECT COUNT(*) FROM wallets WHERE user_id = ?", userID).Scan(&exists)
	if err != nil {
		log.Printf("Database error: %v", err)
		gw.sendError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if exists > 0 {
		gw.sendError(w, http.StatusConflict, "Wallet already exists")
		return
	}

	walletID := fmt.Sprintf("wallet_%d", time.Now().UnixNano())
	address := "7xUV6YR3rZMfExPqZiovQSUxpnHxr2KJJqFg1bFrpump"
	initialBalance := 1000.0

	_, err = gw.db.Exec(`
		INSERT INTO wallets (id, user_id, address, balance, available, locked, network)
		VALUES (?, ?, ?, ?, ?, 0.0, 'solana-mainnet')
	`, walletID, userID, address, initialBalance, initialBalance)

	if err != nil {
		log.Printf("Wallet creation error: %v", err)
		gw.sendError(w, http.StatusInternalServerError, "Failed to create wallet")
		return
	}

	wallet := Wallet{
		ID:        walletID,
		UserID:    userID,
		Address:   address,
		Balance:   initialBalance,
		Available: initialBalance,
		Locked:    0.0,
		Network:   "solana-mainnet",
		CreatedAt: time.Now(),
	}

	log.Printf("Wallet created for user %s: %s", userID, walletID)

	gw.sendJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"wallet":  wallet,
	})
}

func (gw *APIGateway) getWalletBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		gw.sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var wallet Wallet
	err := gw.db.QueryRow(`
		SELECT id, user_id, address, balance, available, locked, network, created_at
		FROM wallets WHERE user_id = ?
	`, userID).Scan(&wallet.ID, &wallet.UserID, &wallet.Address, &wallet.Balance,
		&wallet.Available, &wallet.Locked, &wallet.Network, &wallet.CreatedAt)

	if err == sql.ErrNoRows {
		gw.sendError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	if err != nil {
		log.Printf("Database error: %v", err)
		gw.sendError(w, http.StatusInternalServerError, "Database error")
		return
	}

	gw.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"balance":   wallet.Balance,
		"available": wallet.Available,
		"locked":    wallet.Locked,
		"currency":  "dGPU",
	})
}

// Rental Handlers
func (gw *APIGateway) createRental(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		gw.sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		GPUID         string  `json:"gpu_id"`
		EscrowAmount  float64 `json:"escrow_amount"`
		DurationHours int     `json:"duration_hours"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		gw.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check if GPU exists and is available
	var gpu GPU
	err := gw.db.QueryRow(`
		SELECT id, model, price_per_hour, status
		FROM gpus WHERE id = ?
	`, req.GPUID).Scan(&gpu.ID, &gpu.Model, &gpu.PricePerHour, &gpu.Status)

	if err == sql.ErrNoRows {
		gw.sendError(w, http.StatusNotFound, "GPU not found")
		return
	}

	if gpu.Status != "available" {
		gw.sendError(w, http.StatusConflict, "GPU is not available")
		return
	}

	// Check wallet balance
	var available float64
	err = gw.db.QueryRow("SELECT available FROM wallets WHERE user_id = ?", userID).Scan(&available)
	if err == sql.ErrNoRows {
		gw.sendError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	if available < req.EscrowAmount {
		gw.sendError(w, http.StatusPaymentRequired, "Insufficient balance")
		return
	}

	// Create rental
	rentalID := fmt.Sprintf("rental_%d", time.Now().UnixNano())
	estimatedEnd := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)

	_, err = gw.db.Exec(`
		INSERT INTO rentals (id, gpu_id, user_id, escrow_amount, hourly_rate, status, estimated_end)
		VALUES (?, ?, ?, ?, ?, 'active', ?)
	`, rentalID, req.GPUID, userID, req.EscrowAmount, gpu.PricePerHour, estimatedEnd)

	if err != nil {
		log.Printf("Rental creation error: %v", err)
		gw.sendError(w, http.StatusInternalServerError, "Failed to create rental")
		return
	}

	// Update GPU status
	_, err = gw.db.Exec("UPDATE gpus SET status = 'rented' WHERE id = ?", req.GPUID)
	if err != nil {
		log.Printf("GPU update error: %v", err)
	}

	// Lock funds in wallet
	_, err = gw.db.Exec(`
		UPDATE wallets
		SET available = available - ?, locked = locked + ?
		WHERE user_id = ?
	`, req.EscrowAmount, req.EscrowAmount, userID)

	if err != nil {
		log.Printf("Wallet update error: %v", err)
	}

	rental := Rental{
		ID:           rentalID,
		GPUID:        req.GPUID,
		UserID:       userID,
		EscrowAmount: req.EscrowAmount,
		HourlyRate:   gpu.PricePerHour,
		Status:       "active",
		StartedAt:    time.Now(),
		EstimatedEnd: estimatedEnd,
	}

	log.Printf("Rental created: %s for user %s", rentalID, userID)

	gw.sendJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"rental":  rental,
	})
}

func (gw *APIGateway) listRentals(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		gw.sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	rows, err := gw.db.Query(`
		SELECT r.id, r.gpu_id, g.model, r.escrow_amount, r.hourly_rate, r.status,
		       r.started_at, r.estimated_end
		FROM rentals r
		JOIN gpus g ON r.gpu_id = g.id
		WHERE r.user_id = ?
		ORDER BY r.started_at DESC
		LIMIT 10
	`, userID)

	if err != nil {
		log.Printf("Database error: %v", err)
		gw.sendError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	type RentalWithGPU struct {
		ID            string    `json:"id"`
		GPUID         string    `json:"gpu_id"`
		GPUModel      string    `json:"gpu_model"`
		EscrowAmount  float64   `json:"escrow_amount"`
		HourlyRate    float64   `json:"hourly_rate"`
		Status        string    `json:"status"`
		StartedAt     time.Time `json:"started_at"`
		EstimatedEnd  time.Time `json:"estimated_end"`
		DurationHours float64   `json:"duration_hours"`
		TotalCost     float64   `json:"total_cost"`
	}

	var rentals []RentalWithGPU
	for rows.Next() {
		var rental RentalWithGPU
		err := rows.Scan(&rental.ID, &rental.GPUID, &rental.GPUModel, &rental.EscrowAmount,
			&rental.HourlyRate, &rental.Status, &rental.StartedAt, &rental.EstimatedEnd)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		// Calculate duration and cost
		duration := time.Since(rental.StartedAt)
		rental.DurationHours = duration.Hours()
		rental.TotalCost = rental.DurationHours * rental.HourlyRate

		rentals = append(rentals, rental)
	}

	gw.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"rentals": rentals,
		"total":   len(rentals),
	})
}

// Stats Handler
func (gw *APIGateway) getUserStats(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		gw.sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var stats struct {
		ActiveRentals int     `json:"activeRentals"`
		TotalSpent    float64 `json:"totalSpent"`
		HoursUsed     float64 `json:"hoursUsed"`
		Savings       float64 `json:"savings"`
	}

	// Get active rentals count
	err := gw.db.QueryRow(`
		SELECT COUNT(*) FROM rentals WHERE user_id = ? AND status = 'active'
	`, userID).Scan(&stats.ActiveRentals)

	if err != nil {
		log.Printf("Stats error: %v", err)
	}

	// Get total spent and hours
	rows, err := gw.db.Query(`
		SELECT hourly_rate, started_at, estimated_end, status
		FROM rentals WHERE user_id = ?
	`, userID)

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rate float64
			var start, end time.Time
			var status string
			if err := rows.Scan(&rate, &start, &end, &status); err == nil {
				var duration time.Duration
				if status == "active" {
					duration = time.Since(start)
				} else {
					duration = end.Sub(start)
				}
				hours := duration.Hours()
				stats.HoursUsed += hours
				stats.TotalSpent += hours * rate
			}
		}
	}

	// Calculate savings (10% compared to traditional cloud)
	stats.Savings = stats.TotalSpent * 0.1

	gw.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}

func (gw *APIGateway) health(w http.ResponseWriter, r *http.Request) {
	gw.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  "healthy",
		"service": "api-gateway",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func main() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./dantegpu-gateway.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	gateway := NewAPIGateway(db)

	if err := gateway.initDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	log.Println("Database initialized successfully")

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

	// Logging middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Printf("%s %s - %v", r.Method, r.URL.Path, time.Since(start))
		})
	})

	// Routes
	r.HandleFunc("/health", gateway.health).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/gpus", gateway.listGPUs).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/gpus/{id}", gateway.getGPU).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/wallet/create", gateway.createWallet).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/wallet/balance", gateway.getWalletBalance).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/rentals", gateway.createRental).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/v1/rentals", gateway.listRentals).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/stats/user", gateway.getUserStats).Methods("GET", "OPTIONS")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                                                              ║\n")
	fmt.Printf("║         🚀 DanteGPU API Gateway (REAL)                      ║\n")
	fmt.Printf("║                                                              ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("\n")
	fmt.Printf("✅ Server running on http://localhost:%s\n", port)
	fmt.Printf("✅ Database: %s\n", dbPath)
	fmt.Printf("\n")
	fmt.Printf("📍 Endpoints:\n")
	fmt.Printf("  GET  /health\n")
	fmt.Printf("  GET  /api/v1/gpus\n")
	fmt.Printf("  GET  /api/v1/gpus/{id}\n")
	fmt.Printf("  POST /api/v1/wallet/create\n")
	fmt.Printf("  GET  /api/v1/wallet/balance\n")
	fmt.Printf("  POST /api/v1/rentals\n")
	fmt.Printf("  GET  /api/v1/rentals\n")
	fmt.Printf("  GET  /api/v1/stats/user\n")
	fmt.Printf("\n")

	log.Fatal(http.ListenAndServe(":"+port, r))
}
