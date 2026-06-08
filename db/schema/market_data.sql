CREATE TABLE IF NOT EXISTS market_latest_prices (
    symbol_id TEXT PRIMARY KEY,
    symbol TEXT NOT NULL UNIQUE,
    price_cents BIGINT NOT NULL CHECK (price_cents > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS market_trades (
    trade_id TEXT PRIMARY KEY,
    symbol_id TEXT NOT NULL,
    symbol TEXT NOT NULL,
    price_cents BIGINT NOT NULL CHECK (price_cents > 0),
    quantity_units BIGINT NOT NULL CHECK (quantity_units > 0),
    buy_order_id TEXT NOT NULL,
    sell_order_id TEXT NOT NULL,
    executed_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS market_trades_symbol_id_executed_at_idx
    ON market_trades (symbol_id, executed_at DESC);

CREATE INDEX IF NOT EXISTS market_trades_symbol_executed_at_idx
    ON market_trades (symbol, executed_at DESC);

CREATE TABLE IF NOT EXISTS market_order_book_levels (
    symbol_id TEXT NOT NULL,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL CHECK (side IN ('BID', 'ASK')),
    price_cents BIGINT NOT NULL CHECK (price_cents > 0),
    quantity_units BIGINT NOT NULL CHECK (quantity_units >= 0),
    order_count INTEGER NOT NULL CHECK (order_count >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (symbol_id, side, price_cents)
);

CREATE INDEX IF NOT EXISTS market_order_book_levels_symbol_id_side_price_idx
    ON market_order_book_levels (symbol_id, side, price_cents);

CREATE INDEX IF NOT EXISTS market_order_book_levels_symbol_side_price_idx
    ON market_order_book_levels (symbol, side, price_cents);
