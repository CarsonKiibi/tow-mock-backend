package main

import (
   "database/sql"
   "encoding/json"
   "fmt"
   "log"
   "math"
   "math/rand"
   "net/http"
   "os"
   "strconv"
   "sync"
   "time"

   "github.com/gorilla/mux"
   "github.com/gorilla/websocket"
   _ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

var upgrader = websocket.Upgrader{
   CheckOrigin: func(r *http.Request) bool {
   	return true
   },
}

type GPSData struct {
   JobID     int64   `json:"job_id"`
   DriverID  int64   `json:"driver_id"`
   Latitude  float64 `json:"latitude"`
   Longitude float64 `json:"longitude"`
   Timestamp string  `json:"timestamp"`
   Status    string  `json:"status"`
   Message   string  `json:"message,omitempty"`
}

type ActiveJob struct {
   JobID       int64
   DriverID    int64
   StartLat    float64
   StartLng    float64
   EndLat      float64
   EndLng      float64
   CurrentLat  float64
   CurrentLng  float64
   StartTime   time.Time
   Direction   int // 1 for going to job, -1 for returning
   Completed   bool
   Steps       []GPSCoordinate
   CurrentStep int
}

type GPSCoordinate struct {
   Lat float64
   Lng float64
}

var (
   activeJobs = make(map[int64]*ActiveJob)
   activeMutex = sync.RWMutex{}
   websocketClients = make(map[*websocket.Conn]bool)
   clientsMutex = sync.RWMutex{}
)

func main() {
   if _, err := os.Stat("./database.db"); err == nil {
   	err := os.Remove("./database.db")
   	if err != nil {
   		log.Fatal("Failed to remove existing database:", err)
   	}
   	fmt.Println("Removed existing database")
   }

   var err error
   db, err = sql.Open("sqlite3", "./database.db")
   fmt.Println("Created new database")
   if err != nil {
   	log.Fatal(err)
   }
   defer db.Close()

   // Create tables (your existing code)
   createTables()

   // Seed database with mock data
   seedDatabase(db)

   // Set up routes
   r := mux.NewRouter()
   
   // Jobs endpoints
   r.HandleFunc("/jobs", getJobs).Methods("GET")
   r.HandleFunc("/jobs/available", getAvailableJobs).Methods("GET")
   r.HandleFunc("/jobs", createJob).Methods("POST")
   r.HandleFunc("/jobs/{id}", getJob).Methods("GET")
   r.HandleFunc("/jobs/{id}", updateJob).Methods("PUT")
   r.HandleFunc("/jobs/{id}/assign", assignJobWithValidation).Methods("PUT")
   r.HandleFunc("/jobs/{id}/complete", completeJob).Methods("PUT")
   
   // GPS tracking websocket
   r.HandleFunc("/ws/gps", handleGPSWebSocket).Methods("GET")
   
   // Drivers endpoints
   r.HandleFunc("/drivers", getDrivers).Methods("GET")
   r.HandleFunc("/drivers", createDriver).Methods("POST")
   r.HandleFunc("/drivers/{id}", updateDriver).Methods("PUT")
   r.HandleFunc("/drivers/active", getActiveDrivers).Methods("GET")
   
   // Fleet vehicles endpoints
   r.HandleFunc("/vehicles", getVehicles).Methods("GET")
   r.HandleFunc("/vehicles", createVehicle).Methods("POST")
   r.HandleFunc("/vehicles/{id}", updateVehicle).Methods("PUT")
   r.HandleFunc("/vehicles/active", getActiveVehicles).Methods("GET")
   
   // Invoices endpoints
   r.HandleFunc("/invoices", getInvoices).Methods("GET")
   r.HandleFunc("/invoices", createInvoice).Methods("POST")
   r.HandleFunc("/invoices/{id}", updateInvoice).Methods("PUT")
   r.HandleFunc("/invoices/pending", getPendingInvoices).Methods("GET")
   
   // Payments endpoints
   r.HandleFunc("/payments", createPayment).Methods("POST")
   r.HandleFunc("/invoices/{id}/payments", getPaymentsByInvoice).Methods("GET")
   
   // Impound endpoints
   r.HandleFunc("/impound", getImpoundedVehicles).Methods("GET")
   r.HandleFunc("/impound", addImpoundedVehicle).Methods("POST")
   r.HandleFunc("/impound/{id}/release", releaseVehicle).Methods("PUT")
   r.HandleFunc("/impound/current", getCurrentlyImpounded).Methods("GET")

   // Start GPS simulation goroutine
   go gpsSimulationWorker()
   
   // Apply CORS middleware
   handler := enableCORS(r)
   
   fmt.Println("Server starting on :8080")
   log.Fatal(http.ListenAndServe(":8080", handler))
}

func createTables() {
   _, err := db.Exec(`CREATE TABLE IF NOT EXISTS jobs (
   	id INTEGER PRIMARY KEY AUTOINCREMENT,
   	vehicle_description TEXT NOT NULL,
   	pickup_coordinates TEXT NOT NULL,
   	destination_coordinates TEXT,
   	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
   	job_type TEXT NOT NULL,
   	status TEXT DEFAULT 'pending',
   	assigned_driver_id INTEGER,
   	assigned_vehicle_id INTEGER,
   	completed_at DATETIME,
   	notes TEXT,
   	FOREIGN KEY (assigned_driver_id) REFERENCES drivers(id),
   	FOREIGN KEY (assigned_vehicle_id) REFERENCES fleet_vehicles(id)
   )`)
   if err != nil { log.Fatal(err) }

   _, err = db.Exec(`CREATE TABLE IF NOT EXISTS drivers (
   	id INTEGER PRIMARY KEY AUTOINCREMENT,
   	name TEXT NOT NULL,
   	phone TEXT,
   	license_number TEXT,
   	date_joined DATE DEFAULT CURRENT_DATE,
   	is_active BOOLEAN DEFAULT 1
   )`)
   if err != nil { log.Fatal(err) }

   _, err = db.Exec(`CREATE TABLE IF NOT EXISTS fleet_vehicles (
   	id INTEGER PRIMARY KEY AUTOINCREMENT,
   	vehicle_type TEXT NOT NULL,
   	make TEXT,
   	model TEXT,
   	year INTEGER,
   	license_plate TEXT UNIQUE,
   	capacity_tons REAL,
   	date_acquired DATE DEFAULT CURRENT_DATE,
   	is_active BOOLEAN DEFAULT 1
   )`)
   if err != nil { log.Fatal(err) }

   _, err = db.Exec(`CREATE TABLE IF NOT EXISTS invoices (
   	id INTEGER PRIMARY KEY AUTOINCREMENT,
   	job_id INTEGER NOT NULL,
   	amount DECIMAL(10,2) NOT NULL,
   	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
   	due_date DATE,
   	status TEXT DEFAULT 'pending',
   	customer_name TEXT,
   	customer_phone TEXT,
   	FOREIGN KEY (job_id) REFERENCES jobs(id)
   )`)
   if err != nil { log.Fatal(err) }

   _, err = db.Exec(`CREATE TABLE IF NOT EXISTS payments (
   	id INTEGER PRIMARY KEY AUTOINCREMENT,
   	invoice_id INTEGER NOT NULL,
   	amount DECIMAL(10,2) NOT NULL,
   	payment_method TEXT,
   	paid_at DATETIME DEFAULT CURRENT_TIMESTAMP,
   	reference_number TEXT,
   	FOREIGN KEY (invoice_id) REFERENCES invoices(id)
   )`)
   if err != nil { log.Fatal(err) }

   _, err = db.Exec(`CREATE TABLE IF NOT EXISTS impounded_vehicles (
   	id INTEGER PRIMARY KEY AUTOINCREMENT,
   	job_id INTEGER,
   	vehicle_description TEXT NOT NULL,
   	license_plate TEXT,
   	owner_name TEXT,
   	owner_phone TEXT,
   	impounded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
   	released_at DATETIME,
   	is_currently_impounded BOOLEAN DEFAULT 1,
   	impound_location TEXT,
   	release_fee DECIMAL(10,2),
   	FOREIGN KEY (job_id) REFERENCES jobs(id)
   )`)
   if err != nil { log.Fatal(err) }
}

// Job handlers
func getJobs(w http.ResponseWriter, r *http.Request) {
   rows, err := db.Query(`SELECT id, vehicle_description, pickup_coordinates, destination_coordinates, 
   	created_at, job_type, status, assigned_driver_id, assigned_vehicle_id, completed_at, notes FROM jobs`)
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }
   defer rows.Close()

   var jobs []map[string]interface{}
   for rows.Next() {
   	var id, assignedDriverID, assignedVehicleID sql.NullInt64
   	var vehicleDesc, pickup, destination, jobType, status, notes sql.NullString
   	var createdAt, completedAt sql.NullString

   	err := rows.Scan(&id, &vehicleDesc, &pickup, &destination, &createdAt, &jobType, &status, 
   		&assignedDriverID, &assignedVehicleID, &completedAt, &notes)
   	if err != nil {
   		http.Error(w, err.Error(), http.StatusInternalServerError)
   		return
   	}

   	job := map[string]interface{}{
   		"id": id.Int64,
   		"vehicle_description": vehicleDesc.String,
   		"pickup_coordinates": pickup.String,
   		"destination_coordinates": destination.String,
   		"created_at": createdAt.String,
   		"job_type": jobType.String,
   		"status": status.String,
   		"assigned_driver_id": assignedDriverID.Int64,
   		"assigned_vehicle_id": assignedVehicleID.Int64,
   		"completed_at": completedAt.String,
   		"notes": notes.String,
   	}
   	jobs = append(jobs, job)
   }

   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(jobs)
}

func createJob(w http.ResponseWriter, r *http.Request) {
   var job map[string]interface{}
   if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
   	http.Error(w, err.Error(), http.StatusBadRequest)
   	return
   }

   result, err := db.Exec(`INSERT INTO jobs (vehicle_description, pickup_coordinates, destination_coordinates, job_type, notes) 
   	VALUES (?, ?, ?, ?, ?)`,
   	job["vehicle_description"], job["pickup_coordinates"], job["destination_coordinates"], job["job_type"], job["notes"])
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }

   id, _ := result.LastInsertId()
   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(map[string]int64{"id": id})
}


func completeJob(w http.ResponseWriter, r *http.Request) {
   vars := mux.Vars(r)
   jobID := vars["id"]

   _, err := db.Exec(`UPDATE jobs SET status = 'completed', completed_at = CURRENT_TIMESTAMP WHERE id = ?`, jobID)
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }

   w.WriteHeader(http.StatusOK)
}

// Driver handlers
func getDrivers(w http.ResponseWriter, r *http.Request) {
   rows, err := db.Query(`SELECT id, name, phone, license_number, date_joined, is_active FROM drivers`)
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }
   defer rows.Close()

   var drivers []map[string]interface{}
   for rows.Next() {
   	var id sql.NullInt64
   	var name, phone, license, dateJoined sql.NullString
   	var isActive sql.NullBool

   	err := rows.Scan(&id, &name, &phone, &license, &dateJoined, &isActive)
   	if err != nil {
   		http.Error(w, err.Error(), http.StatusInternalServerError)
   		return
   	}

   	driver := map[string]interface{}{
   		"id": id.Int64,
   		"name": name.String,
   		"phone": phone.String,
   		"license_number": license.String,
   		"date_joined": dateJoined.String,
   		"is_active": isActive.Bool,
   	}
   	drivers = append(drivers, driver)
   }

   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(drivers)
}

func createDriver(w http.ResponseWriter, r *http.Request) {
   var driver map[string]interface{}
   if err := json.NewDecoder(r.Body).Decode(&driver); err != nil {
   	http.Error(w, err.Error(), http.StatusBadRequest)
   	return
   }

   result, err := db.Exec(`INSERT INTO drivers (name, phone, license_number) VALUES (?, ?, ?)`,
   	driver["name"], driver["phone"], driver["license_number"])
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }

   id, _ := result.LastInsertId()
   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func getActiveDrivers(w http.ResponseWriter, r *http.Request) {
   rows, err := db.Query(`SELECT id, name, phone FROM drivers WHERE is_active = 1`)
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }
   defer rows.Close()

   var drivers []map[string]interface{}
   for rows.Next() {
   	var id sql.NullInt64
   	var name, phone sql.NullString

   	err := rows.Scan(&id, &name, &phone)
   	if err != nil {
   		http.Error(w, err.Error(), http.StatusInternalServerError)
   		return
   	}

   	driver := map[string]interface{}{
   		"id": id.Int64,
   		"name": name.String,
   		"phone": phone.String,
   	}
   	drivers = append(drivers, driver)
   }

   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(drivers)
}

// Vehicle handlers
func getVehicles(w http.ResponseWriter, r *http.Request) {
   rows, err := db.Query(`SELECT id, vehicle_type, make, model, year, license_plate, capacity_tons, is_active FROM fleet_vehicles`)
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }
   defer rows.Close()

   var vehicles []map[string]interface{}
   for rows.Next() {
   	var id, year sql.NullInt64
   	var vehicleType, make, model, licensePlate sql.NullString
   	var capacity sql.NullFloat64
   	var isActive sql.NullBool

   	err := rows.Scan(&id, &vehicleType, &make, &model, &year, &licensePlate, &capacity, &isActive)
   	if err != nil {
   		http.Error(w, err.Error(), http.StatusInternalServerError)
   		return
   	}

   	vehicle := map[string]interface{}{
   		"id": id.Int64,
   		"vehicle_type": vehicleType.String,
   		"make": make.String,
   		"model": model.String,
   		"year": year.Int64,
   		"license_plate": licensePlate.String,
   		"capacity_tons": capacity.Float64,
   		"is_active": isActive.Bool,
   	}
   	vehicles = append(vehicles, vehicle)
   }

   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(vehicles)
}

func createVehicle(w http.ResponseWriter, r *http.Request) {
   var vehicle map[string]interface{}
   if err := json.NewDecoder(r.Body).Decode(&vehicle); err != nil {
   	http.Error(w, err.Error(), http.StatusBadRequest)
   	return
   }

   result, err := db.Exec(`INSERT INTO fleet_vehicles (vehicle_type, make, model, year, license_plate, capacity_tons) 
   	VALUES (?, ?, ?, ?, ?, ?)`,
   	vehicle["vehicle_type"], vehicle["make"], vehicle["model"], vehicle["year"], vehicle["license_plate"], vehicle["capacity_tons"])
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }

   id, _ := result.LastInsertId()
   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

// Impound handlers
func getImpoundedVehicles(w http.ResponseWriter, r *http.Request) {
   rows, err := db.Query(`SELECT id, job_id, vehicle_description, license_plate, owner_name, owner_phone, 
   	impounded_at, released_at, is_currently_impounded, impound_location, release_fee FROM impounded_vehicles`)
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }
   defer rows.Close()

   var vehicles []map[string]interface{}
   for rows.Next() {
   	var id, jobID sql.NullInt64
   	var vehicleDesc, licensePlate, ownerName, ownerPhone, impoundLocation sql.NullString
   	var impoundedAt, releasedAt sql.NullString
   	var isCurrentlyImpounded sql.NullBool
   	var releaseFee sql.NullFloat64

   	err := rows.Scan(&id, &jobID, &vehicleDesc, &licensePlate, &ownerName, &ownerPhone, 
   		&impoundedAt, &releasedAt, &isCurrentlyImpounded, &impoundLocation, &releaseFee)
   	if err != nil {
   		http.Error(w, err.Error(), http.StatusInternalServerError)
   		return
   	}

   	vehicle := map[string]interface{}{
   		"id": id.Int64,
   		"job_id": jobID.Int64,
   		"vehicle_description": vehicleDesc.String,
   		"license_plate": licensePlate.String,
   		"owner_name": ownerName.String,
   		"owner_phone": ownerPhone.String,
   		"impounded_at": impoundedAt.String,
   		"released_at": releasedAt.String,
   		"is_currently_impounded": isCurrentlyImpounded.Bool,
   		"impound_location": impoundLocation.String,
   		"release_fee": releaseFee.Float64,
   	}
   	vehicles = append(vehicles, vehicle)
   }

   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(vehicles)
}

func getCurrentlyImpounded(w http.ResponseWriter, r *http.Request) {
   rows, err := db.Query(`SELECT id, vehicle_description, license_plate, owner_name, impounded_at, impound_location 
   	FROM impounded_vehicles WHERE is_currently_impounded = 1`)
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }
   defer rows.Close()

   var vehicles []map[string]interface{}
   for rows.Next() {
   	var id sql.NullInt64
   	var vehicleDesc, licensePlate, ownerName, impoundedAt, location sql.NullString

   	err := rows.Scan(&id, &vehicleDesc, &licensePlate, &ownerName, &impoundedAt, &location)
   	if err != nil {
   		http.Error(w, err.Error(), http.StatusInternalServerError)
   		return
   	}

   	vehicle := map[string]interface{}{
   		"id": id.Int64,
   		"vehicle_description": vehicleDesc.String,
   		"license_plate": licensePlate.String,
   		"owner_name": ownerName.String,
   		"impounded_at": impoundedAt.String,
   		"impound_location": location.String,
   	}
   	vehicles = append(vehicles, vehicle)
   }

   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(vehicles)
}

func addImpoundedVehicle(w http.ResponseWriter, r *http.Request) {
   var vehicle map[string]interface{}
   if err := json.NewDecoder(r.Body).Decode(&vehicle); err != nil {
   	http.Error(w, err.Error(), http.StatusBadRequest)
   	return
   }

   result, err := db.Exec(`INSERT INTO impounded_vehicles (job_id, vehicle_description, license_plate, owner_name, owner_phone, impound_location, release_fee) 
   	VALUES (?, ?, ?, ?, ?, ?, ?)`,
   	vehicle["job_id"], vehicle["vehicle_description"], vehicle["license_plate"], vehicle["owner_name"], 
   	vehicle["owner_phone"], vehicle["impound_location"], vehicle["release_fee"])
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }

   id, _ := result.LastInsertId()
   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func releaseVehicle(w http.ResponseWriter, r *http.Request) {
   vars := mux.Vars(r)
   vehicleID := vars["id"]

   _, err := db.Exec(`UPDATE impounded_vehicles SET is_currently_impounded = 0, released_at = CURRENT_TIMESTAMP WHERE id = ?`, vehicleID)
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }

   w.WriteHeader(http.StatusOK)
}

// Invoice handlers
func createInvoice(w http.ResponseWriter, r *http.Request) {
   var invoice map[string]interface{}
   if err := json.NewDecoder(r.Body).Decode(&invoice); err != nil {
   	http.Error(w, err.Error(), http.StatusBadRequest)
   	return
   }

   result, err := db.Exec(`INSERT INTO invoices (job_id, amount, due_date, customer_name, customer_phone) 
   	VALUES (?, ?, ?, ?, ?)`,
   	invoice["job_id"], invoice["amount"], invoice["due_date"], invoice["customer_name"], invoice["customer_phone"])
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }

   id, _ := result.LastInsertId()
   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

func createPayment(w http.ResponseWriter, r *http.Request) {
   var payment map[string]interface{}
   if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
   	http.Error(w, err.Error(), http.StatusBadRequest)
   	return
   }

   result, err := db.Exec(`INSERT INTO payments (invoice_id, amount, payment_method, reference_number) 
   	VALUES (?, ?, ?, ?)`,
   	payment["invoice_id"], payment["amount"], payment["payment_method"], payment["reference_number"])
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }

   id, _ := result.LastInsertId()
   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

// Placeholder handlers for remaining endpoints
func getJob(w http.ResponseWriter, r *http.Request) { /* implement individual job lookup */ }
func updateJob(w http.ResponseWriter, r *http.Request) { /* implement job updates */ }
func updateDriver(w http.ResponseWriter, r *http.Request) { /* implement driver updates */ }
func updateVehicle(w http.ResponseWriter, r *http.Request) { /* implement vehicle updates */ }
func getInvoices(w http.ResponseWriter, r *http.Request) { /* implement invoice listing */ }
func updateInvoice(w http.ResponseWriter, r *http.Request) { /* implement invoice updates */ }
func getPendingInvoices(w http.ResponseWriter, r *http.Request) { /* implement pending invoices */ }
func getPaymentsByInvoice(w http.ResponseWriter, r *http.Request) { /* implement payments by invoice */ }
func getActiveVehicles(w http.ResponseWriter, r *http.Request) { /* implement active vehicles only */ }

// Get available jobs (pending/unassigned)
func getAvailableJobs(w http.ResponseWriter, r *http.Request) {
   rows, err := db.Query(`SELECT id, vehicle_description, pickup_coordinates, destination_coordinates, 
   	created_at, job_type, notes FROM jobs WHERE status = 'pending' LIMIT 5`)
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }
   defer rows.Close()

   var jobs []map[string]interface{}
   for rows.Next() {
   	var id sql.NullInt64
   	var vehicleDesc, pickup, destination, jobType, notes sql.NullString
   	var createdAt sql.NullString

   	err := rows.Scan(&id, &vehicleDesc, &pickup, &destination, &createdAt, &jobType, &notes)
   	if err != nil {
   		http.Error(w, err.Error(), http.StatusInternalServerError)
   		return
   	}

   	job := map[string]interface{}{
   		"id": id.Int64,
   		"vehicle_description": vehicleDesc.String,
   		"pickup_coordinates": pickup.String,
   		"destination_coordinates": destination.String,
   		"created_at": createdAt.String,
   		"job_type": jobType.String,
   		"status": "pending",
   		"notes": notes.String,
   	}
   	jobs = append(jobs, job)
   }

   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(jobs)
}

// Assign job with validation
func assignJobWithValidation(w http.ResponseWriter, r *http.Request) {
   vars := mux.Vars(r)
   jobIDStr := vars["id"]
   jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
   if err != nil {
   	http.Error(w, "Invalid job ID", http.StatusBadRequest)
   	return
   }

   var assignment map[string]interface{}
   if err := json.NewDecoder(r.Body).Decode(&assignment); err != nil {
   	http.Error(w, err.Error(), http.StatusBadRequest)
   	return
   }

   driverID, ok := assignment["driver_id"]
   if !ok {
   	http.Error(w, "driver_id is required", http.StatusBadRequest)
   	return
   }

   // Verify job exists and is pending
   var jobStatus string
   err = db.QueryRow("SELECT status FROM jobs WHERE id = ?", jobID).Scan(&jobStatus)
   if err == sql.ErrNoRows {
   	http.Error(w, "Job not found", http.StatusNotFound)
   	return
   } else if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }

   if jobStatus != "pending" {
   	http.Error(w, "Job is not available for assignment", http.StatusBadRequest)
   	return
   }

   // Verify driver exists and is active
   var isActive bool
   err = db.QueryRow("SELECT is_active FROM drivers WHERE id = ?", driverID).Scan(&isActive)
   if err == sql.ErrNoRows {
   	http.Error(w, "Driver not found", http.StatusNotFound)
   	return
   } else if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }

   if !isActive {
   	http.Error(w, "Driver is not active", http.StatusBadRequest)
   	return
   }

   // Update job assignment
   _, err = db.Exec(`UPDATE jobs SET assigned_driver_id = ?, status = 'assigned' WHERE id = ?`,
   	driverID, jobID)
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }

   // Start GPS simulation for this job
   startGPSSimulation(jobID, driverID.(float64))

   w.Header().Set("Content-Type", "application/json")
   json.NewEncoder(w).Encode(map[string]string{"status": "assigned"})
}

// WebSocket handler for GPS tracking
func handleGPSWebSocket(w http.ResponseWriter, r *http.Request) {
   conn, err := upgrader.Upgrade(w, r, nil)
   if err != nil {
   	log.Printf("WebSocket upgrade failed: %v", err)
   	return
   }
   defer conn.Close()

   // Add client to active connections
   clientsMutex.Lock()
   websocketClients[conn] = true
   clientsMutex.Unlock()

   // Remove client when connection closes
   defer func() {
   	clientsMutex.Lock()
   	delete(websocketClients, conn)
   	clientsMutex.Unlock()
   }()

   // Keep connection alive
   for {
   	_, _, err := conn.ReadMessage()
   	if err != nil {
   		log.Printf("WebSocket read error: %v", err)
   		break
   	}
   }
}

// Start GPS simulation for a job
func startGPSSimulation(jobID int64, driverIDFloat float64) {
   driverID := int64(driverIDFloat)
   
   // Get job coordinates
   var pickup, destination string
   err := db.QueryRow("SELECT pickup_coordinates, destination_coordinates FROM jobs WHERE id = ?", jobID).Scan(&pickup, &destination)
   if err != nil {
   	log.Printf("Error getting job coordinates: %v", err)
   	return
   }

   // Parse coordinates (assuming format like "lat,lng")
   startLat, startLng := parseCoordinates(pickup)
   endLat, endLng := parseCoordinates(destination)

   // Create active job
   activeJob := &ActiveJob{
   	JobID:       jobID,
   	DriverID:    driverID,
   	StartLat:    startLat,
   	StartLng:    startLng,
   	EndLat:      endLat,
   	EndLng:      endLng,
   	CurrentLat:  startLat,
   	CurrentLng:  startLng,
   	StartTime:   time.Now(),
   	Direction:   1,
   	Completed:   false,
   	CurrentStep: 0,
   }

   // Generate GPS route steps
   activeJob.Steps = generateRoute(startLat, startLng, endLat, endLng)

   // Add to active jobs
   activeMutex.Lock()
   activeJobs[jobID] = activeJob
   activeMutex.Unlock()

   log.Printf("Started GPS simulation for job %d with driver %d", jobID, driverID)
}

// Parse coordinates from string format
func parseCoordinates(coords string) (float64, float64) {
   // For simplicity, using mock coordinates
   // In real implementation, would parse the actual coordinate string
   return 40.7128 + rand.Float64()*0.1 - 0.05, -74.0060 + rand.Float64()*0.1 - 0.05
}

// Generate route between two points
func generateRoute(startLat, startLng, endLat, endLng float64) []GPSCoordinate {
   steps := make([]GPSCoordinate, 0)
   
   // Calculate distance and number of steps
   distance := calculateDistance(startLat, startLng, endLat, endLng)
   numSteps := int(distance / 0.0005) // Approximately 100-300 meters per step
   
   if numSteps < 8 {
   	numSteps = 8 // Minimum steps for 2 minutes journey
   }
   if numSteps > 20 {
   	numSteps = 20 // Maximum steps for 5 minutes journey
   }

   // Generate intermediate points
   for i := 0; i <= numSteps; i++ {
   	progress := float64(i) / float64(numSteps)
   	lat := startLat + (endLat-startLat)*progress
   	lng := startLng + (endLng-startLng)*progress
   	
   	// Add some random variation to make it more realistic
   	lat += (rand.Float64() - 0.5) * 0.001
   	lng += (rand.Float64() - 0.5) * 0.001
   	
   	steps = append(steps, GPSCoordinate{Lat: lat, Lng: lng})
   }

   return steps
}

// Calculate distance between two coordinates (Haversine formula)
func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
   const R = 6371 // Earth's radius in kilometers
   
   dLat := (lat2 - lat1) * math.Pi / 180
   dLng := (lng2 - lng1) * math.Pi / 180
   
   a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dLng/2)*math.Sin(dLng/2)
   c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
   
   return R * c
}

// GPS simulation worker
func gpsSimulationWorker() {
   ticker := time.NewTicker(15 * time.Second)
   defer ticker.Stop()

   log.Println("GPS simulation worker started")

   for {
   	select {
   	case <-ticker.C:
   		processActiveJobs()
   	}
   }
}

// Process all active jobs
func processActiveJobs() {
   activeMutex.Lock()
   defer activeMutex.Unlock()

   for jobID, activeJob := range activeJobs {
   	if activeJob.Completed {
   		continue
   	}

   	// Update GPS position
   	updateJobGPS(activeJob)

   	// Check if job should be completed
   	if activeJob.Direction == 1 && activeJob.CurrentStep >= len(activeJob.Steps)-1 {
   		// Driver has arrived at job location
   		broadcastGPSData(GPSData{
   			JobID:     activeJob.JobID,
   			DriverID:  activeJob.DriverID,
   			Latitude:  activeJob.CurrentLat,
   			Longitude: activeJob.CurrentLng,
   			Timestamp: time.Now().Format(time.RFC3339),
   			Status:    "arrived",
   			Message:   "Driver has arrived at the job location",
   		})
   		
   		// Start return journey
   		activeJob.Direction = -1
   		activeJob.CurrentStep = len(activeJob.Steps) - 1
   		
   	} else if activeJob.Direction == -1 && activeJob.CurrentStep <= 0 {
   		// Driver has completed the job
   		broadcastGPSData(GPSData{
   			JobID:     activeJob.JobID,
   			DriverID:  activeJob.DriverID,
   			Latitude:  activeJob.CurrentLat,
   			Longitude: activeJob.CurrentLng,
   			Timestamp: time.Now().Format(time.RFC3339),
   			Status:    "completed",
   			Message:   "Job completed successfully",
   		})
   		
   		// Mark job as completed in database
   		db.Exec("UPDATE jobs SET status = 'completed', completed_at = CURRENT_TIMESTAMP WHERE id = ?", activeJob.JobID)
   		
   		activeJob.Completed = true
   		delete(activeJobs, jobID)
   		
   	} else {
   		// Send regular GPS update during normal driving
   		broadcastGPSData(GPSData{
   			JobID:     activeJob.JobID,
   			DriverID:  activeJob.DriverID,
   			Latitude:  activeJob.CurrentLat,
   			Longitude: activeJob.CurrentLng,
   			Timestamp: time.Now().Format(time.RFC3339),
   			Status:    getJobStatus(activeJob),
   		})
   	}
   }
}

// Update GPS position for active job
func updateJobGPS(activeJob *ActiveJob) {
   if len(activeJob.Steps) == 0 {
   	return
   }

   // Move to next step
   if activeJob.Direction == 1 && activeJob.CurrentStep < len(activeJob.Steps)-1 {
   	activeJob.CurrentStep++
   } else if activeJob.Direction == -1 && activeJob.CurrentStep > 0 {
   	activeJob.CurrentStep--
   }

   // Update current position
   if activeJob.CurrentStep >= 0 && activeJob.CurrentStep < len(activeJob.Steps) {
   	step := activeJob.Steps[activeJob.CurrentStep]
   	activeJob.CurrentLat = step.Lat
   	activeJob.CurrentLng = step.Lng
   }
}

// Get job status based on direction and progress
func getJobStatus(activeJob *ActiveJob) string {
   if activeJob.Direction == 1 {
   	return "en_route_to_job"
   }
   return "returning_to_base"
}

// Broadcast GPS data to all connected WebSocket clients
func broadcastGPSData(gpsData GPSData) {
   clientsMutex.RLock()
   defer clientsMutex.RUnlock()

   for conn := range websocketClients {
   	err := conn.WriteJSON(gpsData)
   	if err != nil {
   		log.Printf("Error broadcasting GPS data: %v", err)
   		conn.Close()
   		delete(websocketClients, conn)
   	}
   }
   
   // Get driver name and license for logging
   var driverName, licenseNumber string
   err := db.QueryRow("SELECT name, license_number FROM drivers WHERE id = ?", gpsData.DriverID).Scan(&driverName, &licenseNumber)
   if err != nil {
   	driverName = "Unknown"
   	licenseNumber = "Unknown"
   }
   
   log.Printf("GPS data for job %d (%s - %s): %s at %.6f,%.6f", 
   	gpsData.JobID, driverName, licenseNumber, gpsData.Status, gpsData.Latitude, gpsData.Longitude)
}

func enableCORS(next http.Handler) http.Handler {
   return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
   	w.Header().Set("Access-Control-Allow-Origin", "*")
   	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
   	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

   	if r.Method == "OPTIONS" {
   		w.WriteHeader(http.StatusOK)
   		return
   	}

   	next.ServeHTTP(w, r)
   })
}
