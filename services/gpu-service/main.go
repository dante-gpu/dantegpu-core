package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GPUModel struct {
	ID                string                 `json:"id" db:"id"`
	Name              string                 `json:"name" db:"name"`
	Manufacturer      string                 `json:"manufacturer" db:"manufacturer"`
	Architecture      string                 `json:"architecture" db:"architecture"`
	MemoryGB          int                    `json:"memory_gb" db:"memory_gb"`
	MemoryType        string                 `json:"memory_type" db:"memory_type"`
	MemoryBandwidth   float64                `json:"memory_bandwidth_gbps" db:"memory_bandwidth_gbps"`
	CudaCores         int                    `json:"cuda_cores" db:"cuda_cores"`
	TensorCores       *int                   `json:"tensor_cores" db:"tensor_cores"`
	BaseClock         *int                   `json:"base_clock_mhz" db:"base_clock_mhz"`
	BoostClock        *int                   `json:"boost_clock_mhz" db:"boost_clock_mhz"`
	PowerConsumption  *int                   `json:"power_consumption_w" db:"power_consumption_w"`
	PCIeVersion       *string                `json:"pcie_version" db:"pcie_version"`
	Features          map[string]interface{} `json:"features" db:"features"`
	Benchmarks        map[string]interface{} `json:"benchmarks" db:"benchmarks"`
}

type GPUProvider struct {
	ID           string `json:"id" db:"id"`
	Name         string `json:"name" db:"name"`
	Location     string `json:"location" db:"location"`
	ContactEmail string `json:"contact_email" db:"contact_email"`
	Status       string `json:"status" db:"status"`
}

type GPUInstance struct {
	ID           string                 `json:"id" db:"id"`
	Provider     GPUProvider            `json:"provider"`
	Model        GPUModel               `json:"model"`
	InstanceID   string                 `json:"instance_id" db:"instance_id"`
	PricePerHour float64                `json:"price_per_hour" db:"price_per_hour"`
	Status       string                 `json:"status" db:"status"`
	Location     string                 `json:"location" db:"location"`
	Specs        map[string]interface{} `json:"specs" db:"specs"`
}

type GPUService struct {
	db *pgxpool.Pool
}

func NewGPUService(db *pgxpool.Pool) *GPUService {
	return &GPUService{db: db}
}

func (s *GPUService) getGPUs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	status := r.URL.Query().Get("status")
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")

	// Set defaults
	if limit == "" {
		limit = "20"
	}
	if offset == "" {
		offset = "0"
	}

	limitInt, _ := strconv.Atoi(limit)
	offsetInt, _ := strconv.Atoi(offset)

	query := `
		SELECT 
			gi.id, gi.instance_id, gi.price_per_hour, gi.status, gi.location, gi.specs,
			gp.id, gp.name, gp.location, gp.contact_email, gp.status,
			gm.id, gm.name, gm.manufacturer, gm.architecture, gm.memory_gb, 
			gm.memory_type, gm.memory_bandwidth_gbps, gm.cuda_cores, gm.tensor_cores,
			gm.base_clock_mhz, gm.boost_clock_mhz, gm.power_consumption_w, 
			gm.pcie_version, gm.features, gm.benchmarks
		FROM gpu_instances gi
		JOIN gpu_providers gp ON gi.provider_id = gp.id
		JOIN gpu_models gm ON gi.model_id = gm.id
	`

	args := []interface{}{}
	argCount := 0

	if status != "" {
		query += " WHERE gi.status = $" + strconv.Itoa(argCount+1)
		args = append(args, status)
		argCount++
	}

	query += " ORDER BY gi.price_per_hour ASC"
	query += " LIMIT $" + strconv.Itoa(argCount+1) + " OFFSET $" + strconv.Itoa(argCount+2)
	args = append(args, limitInt, offsetInt)

	rows, err := s.db.Query(context.Background(), query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var gpus []GPUInstance
	for rows.Next() {
		var gpu GPUInstance
		var specsJSON []byte
		var featuresJSON []byte
		var benchmarksJSON []byte

		err := rows.Scan(
			&gpu.ID, &gpu.InstanceID, &gpu.PricePerHour, &gpu.Status, &gpu.Location, &specsJSON,
			&gpu.Provider.ID, &gpu.Provider.Name, &gpu.Provider.Location, &gpu.Provider.ContactEmail, &gpu.Provider.Status,
			&gpu.Model.ID, &gpu.Model.Name, &gpu.Model.Manufacturer, &gpu.Model.Architecture, &gpu.Model.MemoryGB,
			&gpu.Model.MemoryType, &gpu.Model.MemoryBandwidth, &gpu.Model.CudaCores, &gpu.Model.TensorCores,
			&gpu.Model.BaseClock, &gpu.Model.BoostClock, &gpu.Model.PowerConsumption,
			&gpu.Model.PCIeVersion, &featuresJSON, &benchmarksJSON,
		)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}

		// Parse JSON fields
		if specsJSON != nil {
			json.Unmarshal(specsJSON, &gpu.Specs)
		}
		if featuresJSON != nil {
			json.Unmarshal(featuresJSON, &gpu.Model.Features)
		}
		if benchmarksJSON != nil {
			json.Unmarshal(benchmarksJSON, &gpu.Model.Benchmarks)
		}

		gpus = append(gpus, gpu)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gpus)
}

func (s *GPUService) getGPU(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gpuID := vars["id"]

	query := `
		SELECT 
			gi.id, gi.instance_id, gi.price_per_hour, gi.status, gi.location, gi.specs,
			gp.id, gp.name, gp.location, gp.contact_email, gp.status,
			gm.id, gm.name, gm.manufacturer, gm.architecture, gm.memory_gb, 
			gm.memory_type, gm.memory_bandwidth_gbps, gm.cuda_cores, gm.tensor_cores,
			gm.base_clock_mhz, gm.boost_clock_mhz, gm.power_consumption_w, 
			gm.pcie_version, gm.features, gm.benchmarks
		FROM gpu_instances gi
		JOIN gpu_providers gp ON gi.provider_id = gp.id
		JOIN gpu_models gm ON gi.model_id = gm.id
		WHERE gi.id = $1
	`

	var gpu GPUInstance
	var specsJSON []byte
	var featuresJSON []byte
	var benchmarksJSON []byte

	err := s.db.QueryRow(context.Background(), query, gpuID).Scan(
		&gpu.ID, &gpu.InstanceID, &gpu.PricePerHour, &gpu.Status, &gpu.Location, &specsJSON,
		&gpu.Provider.ID, &gpu.Provider.Name, &gpu.Provider.Location, &gpu.Provider.ContactEmail, &gpu.Provider.Status,
		&gpu.Model.ID, &gpu.Model.Name, &gpu.Model.Manufacturer, &gpu.Model.Architecture, &gpu.Model.MemoryGB,
		&gpu.Model.MemoryType, &gpu.Model.MemoryBandwidth, &gpu.Model.CudaCores, &gpu.Model.TensorCores,
		&gpu.Model.BaseClock, &gpu.Model.BoostClock, &gpu.Model.PowerConsumption,
		&gpu.Model.PCIeVersion, &featuresJSON, &benchmarksJSON,
	)

	if err != nil {
		http.Error(w, "GPU not found", http.StatusNotFound)
		return
	}

	// Parse JSON fields
	if specsJSON != nil {
		json.Unmarshal(specsJSON, &gpu.Specs)
	}
	if featuresJSON != nil {
		json.Unmarshal(featuresJSON, &gpu.Model.Features)
	}
	if benchmarksJSON != nil {
		json.Unmarshal(benchmarksJSON, &gpu.Model.Benchmarks)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gpu)
}

func (s *GPUService) health(w http.ResponseWriter, r *http.Request) {
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

	gpuService := NewGPUService(db)

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
	r.HandleFunc("/health", gpuService.health).Methods("GET")
	r.HandleFunc("/gpus", gpuService.getGPUs).Methods("GET")
	r.HandleFunc("/gpus/{id}", gpuService.getGPU).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8091"
	}

	fmt.Printf("GPU service starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
