## Quick Start with Docker

### Prerequisites
- Docker and Docker Compose installed on your system

**Build and run manually:**
```bash
# Build the image
docker build -t towing-mock-backend .

# Run the container
docker run -p 8080:8080 towing-mock-backend

# A new database will be instantiated every time (as this is a MOCK api)!
```

### Docker Management

**View real-time logs (including GPS tracking):**
```bash
# With docker-compose
docker compose logs -f

# With manual container (replace container-name)
docker logs -f towing-mock-backend
```

## Complete API Documentation

### Job Management Endpoints

#### `GET /jobs`
Get all jobs in the system.
- **Method**: GET
- **Request Body**: None
- **Response**: Array of job objects
```json
[
  {
    "id": 1,
    "vehicle_description": "2020 Ford F-150 - Black",
    "pickup_coordinates": "123 Main St, Downtown",
    "destination_coordinates": "456 Oak Ave, Midtown",
    "created_at": "2025-09-07T03:05:27Z",
    "job_type": "accident",
    "status": "pending",
    "assigned_driver_id": 0,
    "assigned_vehicle_id": 0,
    "completed_at": "",
    "notes": "Job #1 - accident tow request"
  }
]
```

#### `GET /jobs/available`
Get up to 5 available jobs for assignment (status: pending).
- **Method**: GET  
- **Request Body**: None
- **Response**: Array of available job objects (max 5)
```json
[
  {
    "id": 1,
    "vehicle_description": "2020 Ford F-150 - Black", 
    "pickup_coordinates": "123 Main St, Downtown",
    "destination_coordinates": "456 Oak Ave, Midtown",
    "created_at": "2025-09-07T03:05:27Z",
    "job_type": "accident",
    "status": "pending",
    "notes": "Job #1 - accident tow request"
  }
]
```

#### `POST /jobs`
Create a new towing job.
- **Method**: POST
- **Content-Type**: application/json
- **Request Body**:
```json
{
  "vehicle_description": "2018 Honda Civic - Blue",
  "pickup_coordinates": "789 Pine St, Location",
  "destination_coordinates": "321 Elm St, Destination", 
  "job_type": "breakdown",
  "notes": "Customer called for breakdown assistance"
}
```
- **Response**:
```json
{
  "id": 16
}
```

#### `PUT /jobs/{id}/assign`
Assign a driver to a job. **This automatically starts GPS simulation.**
- **Method**: PUT
- **Content-Type**: application/json
- **URL Parameter**: `id` (job ID)
- **Request Body**:
```json
{
  "driver_id": 1
}
```
- **Response**: 
```json
{
  "status": "assigned"
}
```
- **Error Responses**:
  - 400: "driver_id is required" 
  - 404: "Job not found" or "Driver not found"
  - 400: "Job is not available for assignment" or "Driver is not active"

#### `PUT /jobs/{id}/complete` 
Mark a job as completed manually.
- **Method**: PUT
- **URL Parameter**: `id` (job ID)
- **Request Body**: None
- **Response**: 200 OK

### Driver Management Endpoints

#### `GET /drivers`
Get all drivers in the system.
- **Method**: GET
- **Request Body**: None
- **Response**: Array of driver objects
```json
[
  {
    "id": 1,
    "name": "John Smith",
    "phone": "555-0101", 
    "license_number": "DL123456",
    "date_joined": "2025-09-07T00:00:00Z",
    "is_active": true
  }
]
```

#### `GET /drivers/active`
Get only active drivers.
- **Method**: GET
- **Request Body**: None  
- **Response**: Array of active driver objects (same format as above, filtered)

#### `POST /drivers`
Create a new driver.
- **Method**: POST
- **Content-Type**: application/json
- **Request Body**:
```json
{
  "name": "Jane Doe",
  "phone": "555-0199",
  "license_number": "DL999888"
}
```
- **Response**:
```json
{
  "id": 6
}
```

### Vehicle Management Endpoints

#### `GET /vehicles` 
Get all fleet vehicles.
- **Method**: GET
- **Request Body**: None
- **Response**: Array of vehicle objects
```json
[
  {
    "id": 1,
    "vehicle_type": "Heavy Tow Truck",
    "make": "Peterbilt", 
    "model": "379",
    "year": 2020,
    "license_plate": "TOW001",
    "capacity_tons": 25.0,
    "is_active": true
  }
]
```

#### `GET /vehicles/active`
Get only active vehicles.
- **Method**: GET
- **Request Body**: None
- **Response**: Array of active vehicle objects

#### `POST /vehicles`
Add a new vehicle to the fleet.
- **Method**: POST
- **Content-Type**: application/json
- **Request Body**:
```json
{
  "vehicle_type": "Medium Tow Truck",
  "make": "Ford",
  "model": "F-550", 
  "year": 2023,
  "license_plate": "TOW006",
  "capacity_tons": 10.0
}
```
- **Response**:
```json
{
  "id": 6
}
```

### GPS Tracking WebSocket

#### `GET /ws/gps` 
WebSocket endpoint for real-time GPS tracking updates.
- **Protocol**: WebSocket
- **Connection**: `ws://localhost:8080/ws/gps`
- **Authentication**: None required
- **Message Format**: JSON messages sent every 15 seconds during active jobs

**GPS Message Types:**
1. **En Route**: Driver traveling to job location
```json
{
  "job_id": 1,
  "driver_id": 1, 
  "latitude": 40.7128,
  "longitude": -74.0060,
  "timestamp": "2025-09-07T03:22:07Z",
  "status": "en_route_to_job"
}
```

2. **Arrival**: Driver reached job location  
```json
{
  "job_id": 1,
  "driver_id": 1,
  "latitude": 40.7589,
  "longitude": -73.9851,
  "timestamp": "2025-09-07T03:25:30Z", 
  "status": "arrived",
  "message": "Driver has arrived at the job location"
}
```

3. **Returning**: Driver returning to base
```json
{
  "job_id": 1,
  "driver_id": 1,
  "latitude": 40.7440,
  "longitude": -73.9900,
  "timestamp": "2025-09-07T03:28:15Z",
  "status": "returning_to_base"
}
```

4. **Completed**: Job finished
```json
{
  "job_id": 1,
  "driver_id": 1,
  "latitude": 40.7128,
  "longitude": -74.0060, 
  "timestamp": "2025-09-07T03:32:45Z",
  "status": "completed",
  "message": "Job completed successfully" 
}
```

### Invoice Management Endpoints

#### `GET /invoices`
Get all invoices.
- **Method**: GET
- **Request Body**: None
- **Response**: Array of invoice objects

#### `GET /invoices/pending` 
Get unpaid invoices.
- **Method**: GET
- **Request Body**: None
- **Response**: Array of pending invoice objects

#### `POST /invoices`
Create a new invoice.
- **Method**: POST
- **Content-Type**: application/json
- **Request Body**:
```json
{
  "job_id": 1,
  "amount": 250.00,
  "due_date": "2025-10-07",
  "customer_name": "John Customer", 
  "customer_phone": "555-1234"
}
```
- **Response**:
```json
{
  "id": 11
}
```

### Payment Endpoints

#### `POST /payments`
Record a payment for an invoice.
- **Method**: POST
- **Content-Type**: application/json  
- **Request Body**:
```json
{
  "invoice_id": 1,
  "amount": 250.00,
  "payment_method": "credit_card",
  "reference_number": "REF123456"
}
```
- **Response**:
```json
{
  "id": 5
}
```

#### `GET /invoices/{id}/payments`
Get all payments for a specific invoice.
- **Method**: GET
- **URL Parameter**: `id` (invoice ID)
- **Request Body**: None
- **Response**: Array of payment objects

### Impound Management Endpoints

#### `GET /impound`
Get all impounded vehicles.
- **Method**: GET
- **Request Body**: None
- **Response**: Array of impound records

#### `GET /impound/current`
Get currently impounded vehicles only.
- **Method**: GET
- **Request Body**: None  
- **Response**: Array of currently impounded vehicles

#### `POST /impound`
Add a vehicle to impound.
- **Method**: POST
- **Content-Type**: application/json
- **Request Body**:
```json
{
  "job_id": 1,
  "vehicle_description": "2018 Toyota Camry - Red",
  "license_plate": "ABC123", 
  "owner_name": "Vehicle Owner",
  "owner_phone": "555-5678",
  "impound_location": "City Impound Lot A",
  "release_fee": 300.00
}
```
- **Response**:
```json
{
  "id": 5
}
```

#### `PUT /impound/{id}/release`
Release a vehicle from impound.
- **Method**: PUT
- **URL Parameter**: `id` (impound record ID)
- **Request Body**: None
- **Response**: 200 OK

## GPS Simulation Flow

The GPS simulation provides realistic job progression for frontend development:

1. **Job Assignment**: When a driver is assigned via `PUT /jobs/{id}/assign`, GPS simulation starts automatically
2. **En Route Phase**: 
   - GPS coordinates update every 15 seconds 
   - Status: `"en_route_to_job"`
   - Coordinates move ~100-300 meters per update
   - Duration: 2-5 minutes (randomized)
3. **Arrival**: 
   - System broadcasts `"arrived"` status when driver reaches destination
   - Includes arrival message
4. **Return Journey**: 
   - Driver automatically starts return trip
   - Status: `"returning_to_base"`
   - Retraces original route back to starting point
5. **Completion**: 
   - Status: `"completed"` with completion message
   - Job marked as completed in database
   - GPS simulation ends and cleans up

## Data Model Reference

### Job Status Values
- `"pending"` - Available for assignment
- `"assigned"` - Driver assigned, GPS simulation starting
- `"in_progress"` - Job in progress (legacy status)
- `"completed"` - Job finished

### Job Types
- `"accident"` - Accident recovery
- `"breakdown"` - Vehicle breakdown assistance
- `"police"` - Police-requested tow
- `"parking_violation"` - Parking violation tow
- `"repo"` - Vehicle repossession

### GPS Status Values
- `"en_route_to_job"` - Driver traveling to job location
- `"arrived"` - Driver has arrived at job
- `"returning_to_base"` - Driver returning from completed job
- `"completed"` - Job fully completed

## Frontend Development Guide

This API is designed to support a complete towing management frontend. Key integration points:

### Essential Features for Frontend
1. **Job Dashboard**: Display available jobs from `/jobs/available`
2. **Driver Assignment**: Use dropdown of active drivers from `/drivers/active`
3. **Real-time Tracking**: Connect to WebSocket for live GPS updates
4. **Job Status Updates**: Monitor job progression through GPS status changes
5. **Fleet Management**: Display vehicles and drivers from respective endpoints

### Recommended Frontend Flow
```
1. Load available jobs → GET /jobs/available
2. Show active drivers → GET /drivers/active  
3. User assigns driver → PUT /jobs/{id}/assign
4. Connect to GPS feed → WebSocket /ws/gps
5. Display real-time updates → Parse WebSocket messages
6. Show completion → Handle "completed" status
```

### WebSocket Integration Example
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/gps');

ws.onmessage = function(event) {
  const gpsData = JSON.parse(event.data);
  
  // Update map marker position
  updateDriverLocation(gpsData.job_id, gpsData.latitude, gpsData.longitude);
  
  // Handle status changes
  switch(gpsData.status) {
    case 'en_route_to_job':
      showDriverEnRoute(gpsData.job_id);
      break;
    case 'arrived':
      showDriverArrived(gpsData.job_id, gpsData.message);
      break;
    case 'returning_to_base':
      showDriverReturning(gpsData.job_id);
      break;
    case 'completed':
      showJobCompleted(gpsData.job_id, gpsData.message);
      refreshJobsList(); // Reload available jobs
      break;
  }
};
```

## Testing and Development

### Manual API Testing
```bash
# 1. Get available jobs
curl http://localhost:8080/jobs/available

# 2. Get active drivers  
curl http://localhost:8080/drivers/active

# 3. Assign driver to job (starts GPS simulation)
curl -X PUT -H "Content-Type: application/json" \
  -d '{"driver_id": 1}' \
  http://localhost:8080/jobs/1/assign

# 4. Create new job
curl -X POST -H "Content-Type: application/json" \
  -d '{"vehicle_description": "2018 Honda Civic", "pickup_coordinates": "123 Main St", "destination_coordinates": "456 Oak Ave", "job_type": "breakdown", "notes": "Engine trouble"}' \
  http://localhost:8080/jobs
```

### Monitoring GPS Simulation
View real-time GPS tracking logs:
```bash
# With docker-compose
docker compose logs -f

# With manual container
docker logs -f towing-mock-backend
```

**Sample GPS Log Output:**
```
2025/09/06 20:22:07 GPS data for job 1 (John Smith - DL123456): en_route_to_job at 40.702129,-74.035454
2025/09/06 20:22:22 GPS data for job 1 (John Smith - DL123456): en_route_to_job at 40.704308,-74.033712
2025/09/06 20:22:37 GPS data for job 1 (John Smith - DL123456): arrived at 40.706891,-74.031245
2025/09/06 20:22:52 GPS data for job 1 (John Smith - DL123456): returning_to_base at 40.704308,-74.033712
2025/09/06 20:23:07 GPS data for job 1 (John Smith - DL123456): completed at 40.702129,-74.035454
```

### CORS Support
All endpoints include CORS headers for frontend development:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type`

## Development Without Docker

If you prefer to run locally for development:

1. **Install Go 1.24.6+**
2. **Install dependencies:**
   ```bash
   go mod download
   ```
3. **Run the application:**
   ```bash
   go run .
   ```
4. **Access API at:** `http://localhost:8080`

## Troubleshooting

### Database Issues
The SQLite database is recreated on each startup with fresh mock data. To reset:
```bash
docker compose down
docker compose up --build
```

### Port Conflicts
If port 8080 is in use:
```bash
# Run on different port
docker run -p 8081:8080 towing-mock-backend

# Update frontend connection to http://localhost:8081
```

### WebSocket Connection Issues
- Ensure WebSocket URL uses `ws://` not `http://`
- Check browser console for connection errors
- Verify CORS settings if connecting from different origin
