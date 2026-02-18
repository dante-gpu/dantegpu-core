package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	Name     string  `json:"name"`
	Avatar   *string `json:"avatar"`
	Balance  float64 `json:"balance"`
	Verified bool    `json:"verified"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token"`
	User    User   `json:"user"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type AuthService struct {
	db        *sql.DB
	jwtSecret []byte
}

func NewAuthService(db *sql.DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *AuthService) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		name TEXT NOT NULL,
		avatar_url TEXT,
		balance REAL DEFAULT 0.0,
		verified INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *AuthService) generateToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
	})

	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func (s *AuthService) checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *AuthService) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (s *AuthService) sendError(w http.ResponseWriter, statusCode int, message string) {
	s.sendJSON(w, statusCode, ErrorResponse{
		Success: false,
		Error:   message,
	})
}

func (s *AuthService) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		s.sendError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	var user User
	var passwordHash string
	var verified int

	err := s.db.QueryRow(
		"SELECT id, email, name, avatar_url, balance, verified, password_hash FROM users WHERE email = ?",
		req.Email).Scan(&user.ID, &user.Email, &user.Name, &user.Avatar, &user.Balance, &verified, &passwordHash)

	if err == sql.ErrNoRows {
		s.sendError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if err != nil {
		log.Printf("Database error: %v", err)
		s.sendError(w, http.StatusInternalServerError, "Database error")
		return
	}

	user.Verified = verified == 1

	if !s.checkPassword(req.Password, passwordHash) {
		s.sendError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		log.Printf("Token generation error: %v", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	log.Printf("User logged in: %s (%s)", user.Email, user.ID)

	s.sendJSON(w, http.StatusOK, AuthResponse{
		Success: true,
		Token:   token,
		User:    user,
	})
}

func (s *AuthService) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		s.sendError(w, http.StatusBadRequest, "Email, password, and name are required")
		return
	}

	// Check if user already exists
	var exists int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", req.Email).Scan(&exists)
	if err != nil {
		log.Printf("Database error: %v", err)
		s.sendError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if exists > 0 {
		s.sendError(w, http.StatusConflict, "User already exists")
		return
	}

	// Hash password
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		log.Printf("Password hashing error: %v", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Generate user ID
	userID := fmt.Sprintf("user_%d", time.Now().UnixNano())

	// Create user
	_, err = s.db.Exec(
		`INSERT INTO users (id, email, password_hash, name, balance, verified) 
		 VALUES (?, ?, ?, ?, 1000.0, 0)`,
		userID, req.Email, hashedPassword, req.Name)

	if err != nil {
		log.Printf("User creation error: %v", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	user := User{
		ID:       userID,
		Email:    req.Email,
		Name:     req.Name,
		Balance:  1000.0,
		Verified: false,
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		log.Printf("Token generation error: %v", err)
		s.sendError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	log.Printf("User registered: %s (%s)", user.Email, user.ID)

	s.sendJSON(w, http.StatusCreated, AuthResponse{
		Success: true,
		Token:   token,
		User:    user,
	})
}

func (s *AuthService) profile(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		// Try to extract from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.sendError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// Parse JWT token
		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return s.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			s.sendError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			s.sendError(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		userID, ok = claims["user_id"].(string)
		if !ok {
			s.sendError(w, http.StatusUnauthorized, "Invalid user ID in token")
			return
		}
	}

	var user User
	var verified int
	err := s.db.QueryRow(
		"SELECT id, email, name, avatar_url, balance, verified FROM users WHERE id = ?",
		userID).Scan(&user.ID, &user.Email, &user.Name, &user.Avatar, &user.Balance, &verified)

	if err == sql.ErrNoRows {
		s.sendError(w, http.StatusNotFound, "User not found")
		return
	}

	if err != nil {
		log.Printf("Database error: %v", err)
		s.sendError(w, http.StatusInternalServerError, "Database error")
		return
	}

	user.Verified = verified == 1

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

func (s *AuthService) health(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  "healthy",
		"service": "auth-service",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func main() {
	// Database setup
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./dantegpu.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dantegpu_super_secret_jwt_key_change_in_production_12345"
	}

	authService := NewAuthService(db, jwtSecret)

	// Initialize database schema
	if err := authService.initDB(); err != nil {
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
	r.HandleFunc("/health", authService.health).Methods("GET", "OPTIONS")
	r.HandleFunc("/login", authService.login).Methods("POST", "OPTIONS")
	r.HandleFunc("/register", authService.register).Methods("POST", "OPTIONS")
	r.HandleFunc("/profile", authService.profile).Methods("GET", "OPTIONS")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                                                              ║\n")
	fmt.Printf("║         🔐 DanteGPU Auth Service (REAL)                     ║\n")
	fmt.Printf("║                                                              ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("\n")
	fmt.Printf("✅ Server running on http://localhost:%s\n", port)
	fmt.Printf("✅ Database: %s\n", dbPath)
	fmt.Printf("✅ JWT Secret: %s...\n", jwtSecret[:20])
	fmt.Printf("\n")
	fmt.Printf("📍 Endpoints:\n")
	fmt.Printf("  GET  /health\n")
	fmt.Printf("  POST /login\n")
	fmt.Printf("  POST /register\n")
	fmt.Printf("  GET  /profile\n")
	fmt.Printf("\n")

	log.Fatal(http.ListenAndServe(":"+port, r))
}

