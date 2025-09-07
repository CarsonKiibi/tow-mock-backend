package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func seedDatabase(db *sql.DB) {
	fmt.Println("Seeding database with mock data...")

	// Seed drivers
	drivers := []map[string]interface{}{
		{"name": "John Smith", "phone": "555-0101", "license_number": "DL123456"},
		{"name": "Maria Garcia", "phone": "555-0102", "license_number": "DL789012"},
		{"name": "Mike Johnson", "phone": "555-0103", "license_number": "DL345678"},
		{"name": "Sarah Connor", "phone": "555-0104", "license_number": "DL901234"},
		{"name": "David Wilson", "phone": "555-0105", "license_number": "DL567890"},
	}

	for _, driver := range drivers {
		_, err := db.Exec(`INSERT INTO drivers (name, phone, license_number) VALUES (?, ?, ?)`,
			driver["name"], driver["phone"], driver["license_number"])
		if err != nil {
			log.Printf("Error inserting driver: %v", err)
		}
	}

	// Seed fleet vehicles
	vehicles := []map[string]interface{}{
		{"vehicle_type": "Heavy Tow Truck", "make": "Peterbilt", "model": "379", "year": 2020, "license_plate": "TOW001", "capacity_tons": 25.0},
		{"vehicle_type": "Medium Tow Truck", "make": "Freightliner", "model": "M2", "year": 2019, "license_plate": "TOW002", "capacity_tons": 15.0},
		{"vehicle_type": "Light Tow Truck", "make": "Ford", "model": "F-550", "year": 2021, "license_plate": "TOW003", "capacity_tons": 8.0},
		{"vehicle_type": "Flatbed", "make": "Chevrolet", "model": "Silverado 4500", "year": 2020, "license_plate": "TOW004", "capacity_tons": 12.0},
		{"vehicle_type": "Wrecker", "make": "International", "model": "4300", "year": 2018, "license_plate": "TOW005", "capacity_tons": 20.0},
	}

	for _, vehicle := range vehicles {
		_, err := db.Exec(`INSERT INTO fleet_vehicles (vehicle_type, make, model, year, license_plate, capacity_tons) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			vehicle["vehicle_type"], vehicle["make"], vehicle["model"], vehicle["year"], 
			vehicle["license_plate"], vehicle["capacity_tons"])
		if err != nil {
			log.Printf("Error inserting vehicle: %v", err)
		}
	}

	// Seed jobs
	jobTypes := []string{"police", "breakdown", "accident", "parking_violation", "repo"}
	statuses := []string{"pending", "assigned", "in_progress", "completed"}
	vehicleDescriptions := []string{
		"2018 Honda Civic - Blue",
		"2015 Toyota Camry - Silver",
		"2020 Ford F-150 - Black",
		"2017 BMW 3 Series - White",
		"2019 Chevrolet Malibu - Red",
		"2016 Nissan Altima - Gray",
		"2021 Hyundai Elantra - Blue",
		"2014 Jeep Wrangler - Green",
		"2020 Tesla Model 3 - White",
		"2018 Subaru Outback - Silver",
	}

	locations := []string{
		"123 Main St, Downtown",
		"456 Oak Ave, Midtown",
		"789 Pine Rd, Eastside",
		"321 Elm St, Westside",
		"654 Maple Dr, Northside",
		"987 Cedar Ln, Southside",
		"147 Birch Way, Industrial District",
		"258 Spruce St, Shopping Center",
		"369 Willow Ave, Residential Area",
		"741 Aspen Blvd, Business District",
	}

	for i := 0; i < 15; i++ {
		// Random job data
		vehicleDesc := vehicleDescriptions[rand.Intn(len(vehicleDescriptions))]
		pickup := locations[rand.Intn(len(locations))]
		destination := locations[rand.Intn(len(locations))]
		jobType := jobTypes[rand.Intn(len(jobTypes))]
		status := statuses[rand.Intn(len(statuses))]
		
		// Random assignment (some jobs unassigned)
		var driverID, vehicleID interface{}
		if rand.Float32() > 0.3 { // 70% chance of assignment
			driverID = rand.Intn(5) + 1
			vehicleID = rand.Intn(5) + 1
		}

		// Random completion time for completed jobs
		var completedAt interface{}
		if status == "completed" {
			completedAt = time.Now().Add(-time.Duration(rand.Intn(72)) * time.Hour).Format("2006-01-02 15:04:05")
		}

		notes := fmt.Sprintf("Job #%d - %s tow request", i+1, jobType)

		_, err := db.Exec(`INSERT INTO jobs (vehicle_description, pickup_coordinates, destination_coordinates, 
			job_type, status, assigned_driver_id, assigned_vehicle_id, completed_at, notes) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			vehicleDesc, pickup, destination, jobType, status, driverID, vehicleID, completedAt, notes)
		if err != nil {
			log.Printf("Error inserting job: %v", err)
		}
	}

	// Seed impounded vehicles
	impoundedVehicles := []map[string]interface{}{
		{
			"job_id": 3,
			"vehicle_description": "2019 Honda Accord - Black",
			"license_plate": "ABC123",
			"owner_name": "Robert Brown",
			"owner_phone": "555-1001",
			"impound_location": "City Impound Lot A",
			"release_fee": 250.00,
			"is_currently_impounded": true,
		},
		{
			"job_id": 7,
			"vehicle_description": "2017 Toyota Prius - Silver",
			"license_plate": "XYZ789",
			"owner_name": "Lisa Davis",
			"owner_phone": "555-1002",
			"impound_location": "City Impound Lot B",
			"release_fee": 180.00,
			"is_currently_impounded": true,
		},
		{
			"job_id": 1,
			"vehicle_description": "2020 Ford Explorer - White",
			"license_plate": "DEF456",
			"owner_name": "James Wilson",
			"owner_phone": "555-1003",
			"impound_location": "City Impound Lot A",
			"release_fee": 300.00,
			"is_currently_impounded": false,
			"released_at": time.Now().Add(-time.Duration(48) * time.Hour).Format("2006-01-02 15:04:05"),
		},
		{
			"job_id": 12,
			"vehicle_description": "2016 Chevrolet Cruze - Blue",
			"license_plate": "GHI789",
			"owner_name": "Amanda Johnson",
			"owner_phone": "555-1004",
			"impound_location": "City Impound Lot C",
			"release_fee": 220.00,
			"is_currently_impounded": true,
		},
	}

	for _, vehicle := range impoundedVehicles {
		query := `INSERT INTO impounded_vehicles (job_id, vehicle_description, license_plate, owner_name, 
			owner_phone, impound_location, release_fee, is_currently_impounded`
		values := []interface{}{
			vehicle["job_id"], vehicle["vehicle_description"], vehicle["license_plate"],
			vehicle["owner_name"], vehicle["owner_phone"], vehicle["impound_location"],
			vehicle["release_fee"], vehicle["is_currently_impounded"],
		}

		if releasedAt, exists := vehicle["released_at"]; exists {
			query += ", released_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
			values = append(values, releasedAt)
		} else {
			query += ") VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
		}

		_, err := db.Exec(query, values...)
		if err != nil {
			log.Printf("Error inserting impounded vehicle: %v", err)
		}
	}

	// Seed invoices
	customerNames := []string{"John Doe", "Jane Smith", "Bob Johnson", "Alice Brown", "Charlie Wilson"}
	customerPhones := []string{"555-2001", "555-2002", "555-2003", "555-2004", "555-2005"}

	for i := 1; i <= 10; i++ {
		amount := float64(100 + rand.Intn(400)) // $100-$500
		customerName := customerNames[rand.Intn(len(customerNames))]
		customerPhone := customerPhones[rand.Intn(len(customerPhones))]
		status := []string{"pending", "paid", "overdue"}[rand.Intn(3)]
		dueDate := time.Now().Add(time.Duration(rand.Intn(30)+1) * 24 * time.Hour).Format("2006-01-02")

		result, err := db.Exec(`INSERT INTO invoices (job_id, amount, due_date, status, customer_name, customer_phone) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			i, amount, dueDate, status, customerName, customerPhone)
		if err != nil {
			log.Printf("Error inserting invoice: %v", err)
			continue
		}

		invoiceID, _ := result.LastInsertId()

		// Add payments for some invoices
		if status == "paid" && rand.Float32() > 0.3 {
			paymentMethods := []string{"cash", "credit_card", "check", "bank_transfer"}
			paymentMethod := paymentMethods[rand.Intn(len(paymentMethods))]
			refNumber := fmt.Sprintf("REF%06d", rand.Intn(999999))

			_, err := db.Exec(`INSERT INTO payments (invoice_id, amount, payment_method, reference_number) 
				VALUES (?, ?, ?, ?)`,
				invoiceID, amount, paymentMethod, refNumber)
			if err != nil {
				log.Printf("Error inserting payment: %v", err)
			}
		}
	}

	fmt.Println("Database seeding completed!")
}