# ride-hail 🚗

A real-time, distributed ride-hailing backend inspired by systems like Uber, built with Go, RabbitMQ, WebSockets, and PostgreSQL using Service-Oriented Architecture (SOA).

---

## 🚀 Overview

This project implements a **real-time ride-hailing platform** with multiple cooperating services:

- Passengers create ride requests
- Drivers go online, receive ride offers, and update live locations
- The system matches drivers to passengers, tracks trips in real time, and exposes admin metrics

The platform focuses on **event-driven, high-concurrency backend design** using message queues, WebSockets, and transactional PostgreSQL storage.

---

## 🎯 Learning Objectives

- Advanced **message queue patterns** with RabbitMQ (topic + fanout exchanges)
- **Real-time communication** via WebSockets for passengers and drivers
- **Geospatial processing** for driver proximity and ETA calculation
- **Service-Oriented Architecture** with loosely coupled Go microservices
- **Distributed state management** and audit trail via event sourcing
- **High-concurrency programming** and graceful shutdown in Go

---

## 🧱 Architecture

The system is composed of three main services:

- **Ride Service**
  - Orchestrates the full ride lifecycle
  - Exposes REST API for passengers (`/rides`, `/rides/{id}/cancel`)
  - Publishes ride requests and status changes to RabbitMQ
  - Pushes real-time updates to passengers via WebSocket

- **Driver & Location Service**
  - Manages drivers (online/offline, status, sessions)
  - Runs driver matching based on proximity & vehicle type
  - Consumes ride requests from RabbitMQ
  - Broadcasts driver location updates via fanout exchange
  - Communicates with drivers via WebSocket

- **Admin Service**
  - Exposes monitoring endpoints:
    - `/admin/overview`
    - `/admin/rides/active`
  - Aggregates metrics (active rides, online drivers, revenue, etc.)

**Messaging Layer (RabbitMQ):**

- `ride_topic` (topic)
  - `ride.request.*` → new ride requests
  - `ride.status.*` → ride status changes
- `driver_topic` (topic)
  - `driver.response.*` → driver accept/decline
  - `driver.status.*` → driver availability
- `location_fanout` (fanout)
  - Broadcasts driver location updates to interested consumers

**Persistence Layer (PostgreSQL):**

- `users`, `roles`, `user_status`
- `rides`, `ride_status`, `vehicle_type`
- `coordinates`, `location_history`
- `drivers`, `driver_sessions`
- `ride_events` for full event/audit trail

---

## 🛠 Tech Stack

- **Language:** Go
- **Database:** PostgreSQL
- **Message Broker:** RabbitMQ
- **Real-time:** WebSockets (`gorilla/websocket`)
- **Auth:** JWT (`github.com/golang-jwt/jwt/v5`)
- **DB Driver:** `pgx/v5`
- **Messaging Client:** `github.com/rabbitmq/amqp091-go`
- **Style:** `gofumpt`-compliant codebase

---

## ⚙️ Configuration

Configuration is provided via environment variables with sensible defaults:

```yaml
database:
  host: ${DB_HOST:-localhost}
  port: ${DB_PORT:-5432}
  user: ${DB_USER:-ridehail_user}
  password: ${DB_PASSWORD:-ridehail_pass}
  database: ${DB_NAME:-ridehail_db}

rabbitmq:
  host: ${RABBITMQ_HOST:-localhost}
  port: ${RABBITMQ_PORT:-5672}
  user: ${RABBITMQ_USER:-guest}
  password: ${RABBITMQ_PASSWORD:-guest}

websocket:
  port: ${WS_PORT:-8080}

services:
  ride_service: ${RIDE_SERVICE_PORT:-3000}
  driver_location_service: ${DRIVER_LOCATION_SERVICE_PORT:-3001}
  admin_service: ${ADMIN_SERVICE_PORT:-3004}
```

---

## 🚀 Getting Started

### 1. Prerequisites

- Go (compatible with `gofumpt`)
- PostgreSQL running and accessible
- RabbitMQ server running and accessible

### 2. Apply Database Migrations

Apply the SQL migrations for:

- user, role, and status tables  
- rides, ride events, coordinates  
- driver, session, and location history tables  

(See `migrations/` folder for the full schema.)

### 3. Build the Project

From the project root:

```bash
go build -o ride-hail-system .
```

This builds all services into a single binary: `ride-hail-system`.

### 4. Run the Services

You can run services either via flags, env variables, or a process manager.  
Typical flow:

1. Start PostgreSQL and RabbitMQ
2. Start the **Ride Service**
3. Start the **Driver & Location Service**
4. Start the **Admin Service**
5. Connect WebSocket clients for passengers and drivers

---

## 📡 Core Flows

### Passenger Flow

1. Passenger calls `POST /rides` with pickup and destination
2. Ride Service:
   - validates data
   - calculates estimated fare and distance
   - stores ride with `REQUESTED` status
   - publishes `ride.request.{ride_type}` to `ride_topic`
3. Passenger connects to WebSocket: `ws://{host}/ws/passengers/{passenger_id}`
4. Receives real-time updates:
   - `MATCHED`, `EN_ROUTE`, `ARRIVED`, `IN_PROGRESS`, `COMPLETED`, `CANCELLED`

### Driver Flow

1. Driver sends `POST /drivers/{driver_id}/online`
2. Driver & Location Service:
   - marks driver AVAILABLE
   - tracks live coordinates via `POST /drivers/{driver_id}/location`
3. Driver connects to WebSocket: `ws://{host}/ws/drivers/{driver_id}`
4. Receives `ride_offer` messages and responds with `ride_response`
5. Status changes are propagated through RabbitMQ and WebSocket

---

## 🧪 Logging

All services emit **structured JSON logs** to stdout with:

- `timestamp`, `level`, `service`, `action`, `message`
- `hostname`, `request_id`, `ride_id`
- For errors: `error.msg`, `error.stack`

These logs are suitable for aggregation and tracing across services.

---

## 🔒 Security (High-Level)

- JWT-based authentication (passengers, drivers, admins)
- Role-based access control for APIs
- Coordinate validation and input sanitization
- WebSocket auth within 5 seconds of connection
- Sensitive data excluded from logs

---

## 📌 Status

This project is designed as an **educational, yet production-inspired backend** to practice:

- distributed systems in Go  
- message queues and event-driven architecture  
- geospatial queries and ride lifecycle orchestration  

It can be extended with:

- mobile/web frontends  
- payment integrations  
- more advanced analytics and fraud detection.

---
