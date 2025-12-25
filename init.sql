CREATE DATABASE IF NOT EXISTS solana;

USE solana;

-- Main swaps table
CREATE TABLE IF NOT EXISTS swaps (
    signature String,
    timestamp DateTime64(3),
    pair String,
    token_in String,
    token_out String,
    amount_in Float64,
    amount_out Float64,
    price Float64,
    fee Float64,
    pool String,
    dex String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (pair, timestamp)
SETTINGS index_granularity = 8192;

-- Materialized view for hourly aggregations
CREATE MATERIALIZED VIEW IF NOT EXISTS swaps_hourly
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (pair, dex, hour)
AS SELECT
    pair,
    dex,
    toStartOfHour(timestamp) AS hour,
    count() AS swap_count,
    sum(amount_in) AS total_amount_in,
    sum(amount_out) AS total_amount_out,
    avg(price) AS avg_price,
    min(price) AS min_price,
    max(price) AS max_price,
    sum(fee) AS total_fees
FROM swaps
GROUP BY pair, dex, hour;

-- Index for faster queries
CREATE INDEX IF NOT EXISTS idx_dex ON swaps (dex) TYPE minmax GRANULARITY 4;
CREATE INDEX IF NOT EXISTS idx_timestamp ON swaps (timestamp) TYPE minmax GRANULARITY 4;

