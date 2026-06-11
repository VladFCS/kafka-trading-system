# Kafka Trading System MVP Roadmap

## MVP Scope

Keep the system focused on three services plus one gateway:

- API Gateway
- Order Service
- Matching Engine Service
- Market Data Service

Do not add user service, wallet service, auth, portfolio, or account balances for the MVP.

The MVP flow is:

```text
Create order
-> publish orders.created
-> match compatible orders
-> publish trades.executed and market.price.updated
-> store market read models
-> query market data through HTTP
```

## Phase 1: Stabilize Order Service

Goal: Order Service owns order lifecycle and reliably emits order events.

Tasks:

- Finish `CreateOrder`.
- Finish `GetOrderByID`.
- Finish `CancelOrder`.
- Keep `idempotency_key` for retry-safe order creation.
- Keep outbox publishing for `orders.created`.
- Keep outbox publishing for `orders.updated`.
- Ensure order creation and outbox insert happen in the same DB transaction.
- Ensure order status updates and outbox insert happen in the same DB transaction.
- Add tests for create-order validation.
- Add tests for cancel-order transition.
- Add tests that outbox rows are created with order writes.

Success criteria:

- Order Service can persist an order.
- Order Service can cancel an open order.
- Order Service reliably creates outbox rows for order events.
- `go test ./internal/order-service/...` passes.

## Phase 2: Build Matching Engine Service

Goal: Matching Engine consumes order events and produces trades.

Tasks:

- Create `cmd/matching-engine/main.go`.
- Create `internal/matching-engine/domain`.
- Create `internal/matching-engine/service`.
- Create `internal/matching-engine/events`.
- Consume `orders.created` from Kafka.
- Maintain in-memory order books per symbol.
- Sort BUY orders by highest price first.
- Sort SELL orders by lowest price first.
- Preserve FIFO ordering at the same price.
- Match BUY and SELL when prices cross.
- Emit `trades.executed`.
- Emit `market.price.updated`.
- Emit `orders.updated` when an order becomes filled.
- Ignore duplicate order IDs.

Success criteria:

- Two compatible BUY/SELL orders produce a trade.
- Duplicate `orders.created` events do not duplicate book entries.
- `go test ./internal/matching-engine/...` passes.

## Phase 3: Finish Market Data Service

Goal: Market Data Service builds durable read models from Kafka events.

Tasks:

- Store latest price per symbol.
- Store recent trades.
- Consume `market.price.updated`.
- Consume `trades.executed`.
- Add repository methods:
  - `GetLatestPrice`
  - `UpsertLatestPrice`
  - `InsertTrade`
  - `GetRecentTrades`
- Add service methods:
  - `ApplyMarketPriceUpdated`
  - `ApplyTradeExecuted`
  - `GetLatestPrice`
  - `GetRecentTrades`
- Expose gRPC:
  - `GetLatestPrice`
  - `GetRecentTrades`
- Make trade inserts idempotent by `trade_id`.
- Prevent older price updates from overwriting newer prices.

Success criteria:

- After a trade, latest price is queryable.
- After a trade, recent trades are queryable.
- Duplicate trade events do not create duplicate rows.
- `go test ./internal/market-data-service/...` passes.

## Phase 4: Build API Gateway

Goal: Clients interact with the system through HTTP only.

Tasks:

- Create REST endpoint `POST /orders`.
- Create REST endpoint `GET /orders/{id}`.
- Create REST endpoint `POST /orders/{id}/cancel`.
- Create REST endpoint `GET /market/{symbol}/price`.
- Create REST endpoint `GET /market/{symbol}/trades`.
- Call Order Service through gRPC.
- Call Market Data Service through gRPC.
- Keep business logic out of the gateway.
- Add request validation and clear HTTP error mapping.

Success criteria:

- Client can create orders over HTTP.
- Client can cancel orders over HTTP.
- Client can query latest price over HTTP.
- Client can query recent trades over HTTP.

## Phase 5: Docker Compose MVP

Goal: Local development is easy to run end-to-end.

Tasks:

- Add PostgreSQL service.
- Add Kafka service.
- Add Order Service.
- Add Matching Engine Service.
- Add Market Data Service.
- Add API Gateway.
- Ensure migrations can be applied locally.
- Ensure service configuration works through environment variables.

Success criteria:

```bash
docker compose up
```

Then this manual flow works:

```text
POST /orders BUY
POST /orders SELL
GET /market/BTC-USD/price
GET /market/BTC-USD/trades
```

## Phase 6: Minimum Reliability

Goal: Cover the main distributed-system failure modes without overbuilding.

Tasks:

- Outbox publisher retries failed publishes.
- Outbox publisher marks events published only after Kafka succeeds.
- Kafka consumers commit offsets only after successful processing.
- Matching Engine ignores duplicate order IDs.
- Market Data Service ignores duplicate trade IDs.
- Use Kafka key `symbol` for market/order events that need per-symbol ordering.
- Add structured logs with:
  - `event_id`
  - `order_id`
  - `trade_id`
  - `symbol`
  - Kafka topic/partition/offset where useful

Success criteria:

- Retried order creation does not create duplicate orders.
- Retried Kafka messages do not corrupt Matching Engine state.
- Retried trade events do not duplicate Market Data rows.
- Logs make async flow debuggable.

## MVP Done Definition

The MVP is done when this full flow works:

```text
1. Client creates BUY order through API Gateway.
2. Client creates matching SELL order through API Gateway.
3. Order Service stores both orders and emits orders.created.
4. Matching Engine consumes orders.created and matches the orders.
5. Matching Engine emits trades.executed and market.price.updated.
6. Market Data Service stores the trade and latest price.
7. Client queries market price and recent trades through API Gateway.
```

Everything else is post-MVP.
