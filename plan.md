# Master Implementation Plan: Pulse

**Status:** Ready to Build
**Pattern:** Event-Driven / Outbox Pattern / Shared Monorepo
**Workspace:** `go.work` (Multi-module)

---

## 1. Data Models & Contracts

Before writing logic, we define the shape of our data.

### A. Database Schema (Postgres)

We need two tables in the Gateway’s database to satisfy the **Transactional Outbox** pattern.

```sql
-- Table: orders
CREATE TABLE orders (
    id UUID PRIMARY KEY,
    user_id VARCHAR(50),
    pickup_h3 VARCHAR(15), -- H3 Index (Res 9)
    dropoff_h3 VARCHAR(15),
    status VARCHAR(20), -- PENDING, COMPLETED
    created_at TIMESTAMP
);

-- Table: outbox_events
-- The Poller will lock rows here, send to Kafka, then delete/mark sent.
CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    aggregate_id UUID,     -- Reference to order_id
    aggregate_type VARCHAR(50), -- e.g., "ORDER"
    payload BYTEA,         -- Protobuf binary data
    topic VARCHAR(50),     -- e.g., "demand.orders"
    created_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'PENDING'
);
```

### B. Redis Data Structures

The Engine writes here; The Gateway reads here.

* **Key:** `surge:{h3_index}` (e.g., `surge:8928308280fffff`)
* **Value:** JSON or Protobuf containing `{ "multiplier": 1.5, "updated_at": 1704200000 }`
* **TTL:** 60 seconds (Data expires if no new events come in).

### C. Protobuf Definition (`libs/shared/pb/pulse.proto`)

This defines the contract between Gateway and Engine.

```protobuf
message OrderCreatedEvent {
  string order_id = 1;
  string pickup_h3 = 2; // Resolution 9
  string user_id = 3;
  int64 timestamp = 4;
}
```

---

## 2. Phase-by-Phase Execution

We will build this in **6 Milestones**. We will complete one, verify it, and move to the next.

### Phase 1: The Foundation & Infrastructure

* **Goal:** A running local cloud environment and shared library compilation.
* **Tasks:**
    1. Setup `docker-compose.yml`: Postgres, Redis, Kafka, Zookeeper, OTEL Collector, Jaeger/Tempo, Prometheus/Mimir.
    2. Setup `libs/shared`: Database connection helpers, Logger (Slog/Zap), and Configuration loader.
    3. Setup `libs/telemetry`: OpenTelemetry TracerProvider init.
    4. Setup `protoc`: Generate Go code from `.proto` files into `libs/shared/pb`.

### Phase 2: Gateway & The Outbox

* **Goal:** Receive an HTTP request and reliably persist the intention.
* **Tasks:**
    1. **API:** Implement `POST /api/v1/orders` (Gin).
    2. **Transactions:** Write logic to insert `Order` + `OutboxEvent` in a single SQL transaction.
    3. **The Poller:** Create a background goroutine in Gateway that:
        * `SELECT * FROM outbox_events WHERE status='PENDING' FOR UPDATE SKIP LOCKED`
        * Produces to Kafka (`demand.orders`).
        * Deletes/Updates the row on success.

### Phase 3: The Engine (Stream Processing)

* **Goal:** Calculate Surge Pricing based on incoming stream.
* **Tasks:**
    1. **Consumer:** Setup a Kafka Consumer Group (`pulse-engine-group`).
    2. **State:** Implement the "Sliding Window" logic.
        * *Simplification for V1:* Store "Request Counts" in a separate Redis key (`counts:{h3}`) with a TTL, or keep in-memory maps if replicas = 1. Let's use **Redis** for state to allow scaling the Engine later.
    3. **Math:** Implement: `Multiplier = 1.0 + min(2.5, (Demand / (Supply * Elasticity)))`.
    4. **Writer:** Update `surge:{h3}` in Redis.

### Phase 4: Closing the Loop (Pricing)

* **Goal:** The Gateway uses the calculated data.
* **Tasks:**
    1. **API:** Implement `GET /api/v1/quote?lat=...&lng=...`.
    2. **Logic:** Convert Lat/Lng $\to$ H3 (Res 9).
    3. **Fetch:** `GET surge:{h3}` from Redis.
    4. **Calc:** Apply `(Base + Dist*Rate) * Multiplier`.

### Phase 5: Observability & Resilience

* **Goal:** Production readiness.
* **Tasks:**
    1. **Tracing:** Ensure `TraceID` propagates from HTTP Headers $\to$ Postgres $\to$ Kafka Headers $\to$ Engine.
    2. **Metrics:** Expose Prometheus metrics (`orders_placed_total`, `surge_calculation_duration`).
    3. **Resilience:** Add a Circuit Breaker in Gateway. If Redis is down, return `Multiplier = 1.0`.

### Phase 6: The "Matrix" (Simulation)

* **Goal:** Watch the city come alive.
* **Tasks:**
    1. Build `scripts/simulator`.
    2. Simulate 500 "Drivers" (Supply) updating locations (pushes to `supply.updates` topic).
    3. Simulate 1000 "Riders" (Demand) placing orders.
    4. Visualize the surge map (Optional: Simple HTML/JS frontend polling Redis).
