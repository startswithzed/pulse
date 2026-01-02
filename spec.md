# Project Specification: Pulse

**Version:** 1.0 (2026 Edition)

**Objective:** To design and implement a high-concurrency, event-driven urban logistics engine that masters distributed state, sub-millisecond pricing, and deep observability.

---

## 1. The Story: "The Heartbeat of the City"

**Pulse** is the operating system for a hyper-local ride-hailing and logistics fleet. It manages the delicate balance between rider demand and driver supply across a massive urban grid.

* **The Mission:** Ensure every rider gets a quote in <100ms and every driver is incentivized to move toward "Pulse Points" (high-demand zones) before the surge even happens.
* **The Problem:** Traditional static pricing leads to "Ghost Surges" (demand without supply) and "Supply Deserts." Pulse uses real-time stream processing to prevent these imbalances.

---

## 2. Service Architecture & Monorepo Strategy

We will use a **Modern Go Monorepo** structure. This allows us to share business logic and infrastructure helpers (Postgres, Kafka, OTEL) while maintaining strict separation between the API and the background workers.

### Service Definitions

1. **`pulse-gateway` (The Ingress):** A high-performance **Gin**-based API. It handles rider quote requests, driver heartbeats, and order placements. It prioritizes low latency and uses Redis for immediate state lookups.
2. **`pulse-engine` (The Brain):** The background worker. It consumes high-volume Kafka streams (`demand.events`, `supply.updates`) to recalculate surge multipliers and logistics optimizations.

### Folder Structure

```text
/pulse
├── apps/
│   ├── gateway/            # pulse-gateway (Gin API)
│   │   └── main.go
│   └── engine/             # pulse-engine (Kafka Worker)
│       └── main.go
├── libs/                   # The "Common Code" Heart
│   ├── shared/
│   │   ├── domain/         # Pure business logic & interfaces (H3, Surge math)
│   │   ├── infra/          # Shared Postgres, Redis, & Kafka drivers
│   │   └── pb/             # Protobuf/Contract definitions (if used)
│   └── telemetry/          # Shared OpenTelemetry & LGTM configuration
├── deployments/            # Docker Compose (Dev) & K8s (Prod)
├── scripts/                # The "City Simulator" & Load Generators
└── go.mod                  # Single workspace for all apps/libs

```

---

## 3. The Technical Stack

| Component | Choice | Justification |
| --- | --- | --- |
| **Language** | Go 1.24+ | Native concurrency and minimal footprint. |
| **API** | **Gin** | Standard-bearer for high-throughput Go APIs. |
| **Messaging** | **Kafka** | Industry standard for reliable, high-volume event streaming. |
| **Indexing** | **Uber H3** | Hexagonal grid system for precise spatial surge logic. |
| **Database** | **Postgres** | Source of truth for orders, audits, and user state. |
| **Cache** | **Redis Stack** | Sub-millisecond lookup for hot zones and driver locations. |
| **Observability** | **LGTM Stack** | Loki (Logs), Grafana (UI), Tempo (Traces), Mimir (Metrics). |
| **Telemetry** | **OpenTelemetry** | Global context propagation from API to Kafka to Worker. |

---

## 4. Core Business Logic (The "Brain")

The complexity of Pulse lies in the **Spatio-Temporal Surge Algorithm**.

1. **Hexagonal Aggregation:** Every coordinate (Lat/Lng) is mapped to an **H3 Hexagon**.
2. **Sliding Window Demand:** `pulse-gateway` records every quote request in a Redis Sorted Set. `pulse-engine` calculates the density of requests in that hexagon over the last 60 seconds.
3. **The Multiplier:**

4. **Transactional Outbox:** When an order is created, we use a Postgres transaction to save the `order` and the `outbox_event`. A separate process ensures the event is pushed to Kafka, preventing data loss.

---

## 5. Advanced Production Patterns

To make this truly "Advanced," we are building in these enterprise-grade features:

* **Idempotency:** The `pulse-engine` will use a **Postgres-backed Idempotency Guard** to ensure no Kafka message is processed twice (e.g., charging a customer twice).
* **Circuit Breakers:** If the `pulse-engine` lags, the `pulse-gateway` will trip a circuit breaker and serve "Standard Pricing" from Redis to keep the system responsive.
* **Graceful Shutdown:** All services will listen for OS signals to drain Kafka consumers and close DB connections cleanly.
* **Auto-Documentation:** We will use **Swag/Swagger** annotations within Gin to auto-generate the API spec at `/swagger/index.html`.

---

## 6. Testing & Simulation ("The Matrix")

We don't just "test" Pulse; we simulate a city.

* **Testcontainers:** Every integration test will spin up a real, ephemeral Kafka and Postgres instance in Docker.
* **Traffic Generator:** A dedicated Go script in `/scripts/simulator` that simulates 1,000 drivers moving through a virtual map and 500 riders requesting rides.
* **Chaos Testing:** We will use **Toxiproxy** to drop Kafka partitions and ensure the `pulse-gateway` remains healthy.

---

## 7. Observability (The LGTM Vision)

1. **Traces (Tempo):** Follow a ride request from the moment the user taps "Quote" to the moment the `pulse-engine` updates the price.
2. **Logs (Loki):** High-context logs that include the `trace_id` automatically.
3. **Metrics (Mimir):** Real-time dashboards showing "Rides per Second" and "Average Surge Multiplier per Hexagon."
