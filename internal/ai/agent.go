package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// AgentConfig holds configuration for the AI agent.
type AgentConfig struct {
	// ClickHouse connection settings.
	ClickHouseAddr     string
	ClickHouseDatabase string
	ClickHouseUsername string
	ClickHousePassword string

	// OpenRouter / LLM settings.
	OpenRouterAPIKey string
	// Model name as understood by OpenRouter, e.g. "openai/gpt-4.1-mini".
	Model string

	Logger *logrus.Logger
}

// Agent provides NL→SQL over the swaps table using an LLM and ClickHouse.
type Agent struct {
	llm    llms.Model
	db     *sql.DB
	logger *logrus.Logger
}

// NewAgent creates a new Agent with its own ClickHouse and LLM clients.
func NewAgent(ctx context.Context, cfg AgentConfig) (*Agent, error) {
	if cfg.Logger == nil {
		cfg.Logger = logrus.New()
	}

	if cfg.OpenRouterAPIKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY is required")
	}

	if cfg.Model == "" {
		// Sensible default OpenRouter model (can be overridden by caller).
		cfg.Model = "openai/gpt-4.1-mini"
	}

	// Initialise LLM backed by OpenRouter (OpenAI-compatible API).
	llm, err := openai.New(
		openai.WithToken(cfg.OpenRouterAPIKey),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel(cfg.Model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenRouter LLM: %w", err)
	}

	// Create ClickHouse *sql.DB using the stdlib wrapper.
	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{cfg.ClickHouseAddr},
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouseDatabase,
			Username: cfg.ClickHouseUsername,
			Password: cfg.ClickHousePassword,
		},
	})

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse from AI agent: %w", err)
	}

	cfg.Logger.WithFields(logrus.Fields{
		"addr":     cfg.ClickHouseAddr,
		"database": cfg.ClickHouseDatabase,
		"model":    cfg.Model,
	}).Info("initialized AI agent")

	return &Agent{
		llm:    llm,
		db:     db,
		logger: cfg.Logger,
	}, nil
}

// Close closes underlying resources.
func (a *Agent) Close() error {
	if a.db != nil {
		a.logger.Debug("closing AI agent ClickHouse connection")
		return a.db.Close()
	}
	return nil
}

// AskResult is the structured result of an Ask call.
type AskResult struct {
	SQL    string
	Answer string
}

// Ask takes a natural language question, generates SQL, executes it, and summarises the result.
func (a *Agent) Ask(ctx context.Context, question string) (*AskResult, error) {
	sqlQuery, err := a.generateSQL(ctx, question)
	if err != nil {
		return nil, err
	}

	rowsJSON, err := a.runQuery(ctx, sqlQuery)
	if err != nil {
		return nil, err
	}

	answer, err := a.summariseResult(ctx, question, sqlQuery, rowsJSON)
	if err != nil {
		return nil, err
	}

	return &AskResult{
		SQL:    sqlQuery,
		Answer: answer,
	}, nil
}

// generateSQL asks the LLM to produce a safe SELECT query over solana.swaps.
func (a *Agent) generateSQL(ctx context.Context, question string) (string, error) {
	prompt := fmt.Sprintf(`
You are an expert ClickHouse SQL generator.

Use ONLY the following table:
%s

Rules:
- Return a single SELECT query in ClickHouse SQL.
- Do NOT include any explanation or comments, only the SQL.
- The table is solana.swaps.
- Use timestamp for time filtering.
- Use aggregate functions like sum, avg, count when appropriate.
- If user asks for \"top\" or \"biggest\" something, use ORDER BY ... DESC and LIMIT.
- Never modify data: no INSERT, UPDATE, DELETE, DROP, ALTER, CREATE, TRUNCATE.

User question:
%s
`, swapsSchemaDescription, question)

	resp, err := llms.GenerateFromSinglePrompt(
		ctx,
		a.llm,
		prompt,
		llms.WithMaxTokens(512),
	)
	if err != nil {
		return "", fmt.Errorf("LLM SQL generation failed: %w", err)
	}

	sqlQuery := sanitizeSQL(resp)
	if err := validateSQL(sqlQuery); err != nil {
		return "", err
	}

	a.logger.WithField("sql", sqlQuery).Debug("generated SQL from question")
	return sqlQuery, nil
}

// runQuery executes the generated SQL and encodes results as JSON.
func (a *Agent) runQuery(ctx context.Context, sqlQuery string) (string, error) {
	rows, err := a.db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return "", fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get columns: %w", err)
	}

	var out []map[string]any
	for rows.Next() {
		values := make([]any, len(cols))
		dest := make([]any, len(cols))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}

		rowMap := make(map[string]any, len(cols))
		for i, col := range cols {
			rowMap[col] = values[i]
		}
		out = append(out, rowMap)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("row iteration error: %w", err)
	}

	data, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("failed to marshal rows to JSON: %w", err)
	}

	return string(data), nil
}

// summariseResult asks the LLM to answer the question given SQL + JSON results.
func (a *Agent) summariseResult(ctx context.Context, question, sqlQuery, rowsJSON string) (string, error) {
	prompt := fmt.Sprintf(`
You are a helpful assistant analysing Solana DEX swap analytics.

User question:
%s

SQL that was executed:
%s

Query results in JSON (array of objects, can be empty):
%s

Instructions:
- If the result set is empty, say that no data was found for the question.
- Otherwise, answer the question concisely using bullet points and short sentences.
- Include key numbers (volumes, counts, prices) rounded reasonably.
- Do not restate the raw JSON.
`, question, sqlQuery, rowsJSON)

	resp, err := llms.GenerateFromSinglePrompt(
		ctx,
		a.llm,
		prompt,
		llms.WithMaxTokens(512),
	)
	if err != nil {
		return "", fmt.Errorf("LLM summarisation failed: %w", err)
	}

	return strings.TrimSpace(resp), nil
}

// sanitizeSQL strips code fences and trailing semicolons from the LLM output.
func sanitizeSQL(s string) string {
	s = strings.TrimSpace(s)

	// Remove ``` blocks if present.
	if strings.HasPrefix(s, "```") {
		// Trim the prefix "```" or "```sql"
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSpace(s)
		if strings.HasPrefix(strings.ToLower(s), "sql") {
			s = s[3:]
		}
	}
	s = strings.TrimSpace(s)
	if strings.HasPrefix(strings.ToLower(s), "sql") {
		s = s[3:]
	}
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "```"); idx >= 0 {
		s = s[:idx]
	}

	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ";")
	return strings.TrimSpace(s)
}

// validateSQL enforces a conservative safety policy for generated SQL.
func validateSQL(s string) error {
	if s == "" {
		return fmt.Errorf("empty SQL generated by LLM")
	}

	upper := strings.ToUpper(strings.TrimSpace(s))

	if !strings.HasPrefix(upper, "SELECT") {
		return fmt.Errorf("only SELECT queries are allowed, got: %s", upper[:min(20, len(upper))])
	}

	disallowed := []string{
		"INSERT ", "UPDATE ", "DELETE ", "DROP ", "ALTER ", "TRUNCATE ",
		"CREATE ", "RENAME ", "ATTACH ", "DETACH ",
	}
	for _, kw := range disallowed {
		if strings.Contains(upper, kw) {
			return fmt.Errorf("disallowed SQL keyword %q in generated query", kw)
		}
	}

	if strings.Contains(s, ";") {
		return fmt.Errorf("multiple statements or semicolons are not allowed")
	}

	if !strings.Contains(upper, "FROM SWAPS") && !strings.Contains(upper, "FROM SOLANA.SWAPS") {
		return fmt.Errorf("query must target solana.swaps table")
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
