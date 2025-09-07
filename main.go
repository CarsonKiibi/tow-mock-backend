package main

import (
   "database/sql"
   "encoding/json"
   "fmt"
   "log"
   "net/http"

   "os"

   "github.com/gorilla/mux"
   _ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

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
   r.HandleFunc("/jobs", createJob).Methods("POST")
   r.HandleFunc("/jobs/{id}", getJob).Methods("GET")
   r.HandleFunc("/jobs/{id}", updateJob).Methods("PUT")
   r.HandleFunc("/jobs/{id}/assign", assignJob).Methods("PUT")
   r.HandleFunc("/jobs/{id}/complete", completeJob).Methods("PUT")
   
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

func assignJob(w http.ResponseWriter, r *http.Request) {
   vars := mux.Vars(r)
   jobID := vars["id"]

   var assignment map[string]interface{}
   if err := json.NewDecoder(r.Body).Decode(&assignment); err != nil {
   	http.Error(w, err.Error(), http.StatusBadRequest)
   	return
   }

   _, err := db.Exec(`UPDATE jobs SET assigned_driver_id = ?, assigned_vehicle_id = ?, status = 'assigned' WHERE id = ?`,
   	assignment["driver_id"], assignment["vehicle_id"], jobID)
   if err != nil {
   	http.Error(w, err.Error(), http.StatusInternalServerError)
   	return
   }

   w.WriteHeader(http.StatusOK)
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
