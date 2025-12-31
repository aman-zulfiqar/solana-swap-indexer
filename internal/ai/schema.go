package ai

// swapsSchemaDescription describes the ClickHouse schema used for NL→SQL prompting.
//
// Keeping it in sync with the actual ClickHouse table definition in init.sql.
const swapsSchemaDescription = `
Database: solana
Table: swaps

Columns:
  - signature  String        -- Solana transaction signature (unique id)
  - timestamp  DateTime      -- Block time of the swap (UTC)
  - pair       String        -- Trading pair, e.g. "SOL/USDC"
  - token_in   String        -- Symbol of token sold by the user
  - token_out  String        -- Symbol of token bought by the user
  - amount_in  Float64       -- Amount of token_in
  - amount_out Float64       -- Amount of token_out
  - price      Float64       -- Implied price: amount_out / amount_in (token_out per token_in)
  - fee        Float64       -- Protocol fee rate (e.g. 0.0025)
  - pool       String        -- Pool identifier (e.g. "RaydiumAMM")
  - dex        String        -- DEX name (e.g. "Raydium")

Notes:
  - Larger amount_out generally means larger volume in token_out.
  - For volume calculations you can SUM(amount_out) or SUM(amount_in) depending on the unit you care about.
  - Time filters should use timestamp, e.g. timestamp >= now() - INTERVAL 24 HOUR.
`
