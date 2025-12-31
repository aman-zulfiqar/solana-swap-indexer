package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/ai"
	"github.com/aman-zulfiqar/solana-swap-indexer/internal/config"

	"github.com/sirupsen/logrus"
)

func main() {
	// Flags
	queryFlag := flag.String("q", "", "Run a single natural language query and exit")
	modelFlag := flag.String("model", "openai/gpt-4.1-mini", "OpenRouter model name")
	flag.Parse()

	// Logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logger.SetLevel(logrus.InfoLevel)

	// Config
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		logger.WithError(err).Fatal("invalid configuration")
	}
	if cfg.OpenRouterAPIKey == "" {
		logger.Fatal("OPENROUTER_API_KEY is required for the AI agent. Please set it in your environment or config.")
	}

	// Context + signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down AI agent...")
		cancel()
	}()

	// Agent
	agent, err := ai.NewAgent(ctx, ai.AgentConfig{
		ClickHouseAddr:     cfg.ClickHouseAddr,
		ClickHouseDatabase: cfg.ClickHouseDatabase,
		ClickHouseUsername: cfg.ClickHouseUsername,
		ClickHousePassword: cfg.ClickHousePassword,
		OpenRouterAPIKey:   cfg.OpenRouterAPIKey,
		Model:              *modelFlag,
		Logger:             logger,
	})
	if err != nil {
		logger.WithError(err).Fatal("failed to create AI agent")
	}
	defer agent.Close()

	// Single-shot mode
	if *queryFlag != "" {
		if err := runSingle(ctx, agent, *queryFlag); err != nil {
			logger.WithError(err).Fatal("query failed")
		}
		return
	}

	// REPL mode
	runREPL(ctx, agent)
}

func runSingle(ctx context.Context, agent *ai.Agent, q string) error {
	res, err := agent.Ask(ctx, q)
	if err != nil {
		return err
	}

	fmt.Printf("SQL:\n%s\n\n", res.SQL)
	fmt.Printf("Answer:\n%s\n", res.Answer)
	return nil
}

func runREPL(ctx context.Context, agent *ai.Agent) {
	fmt.Println("Solana Swap AI Agent (NL → ClickHouse SQL)")
	fmt.Println("Type your question and press Enter. Empty line to exit.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		q, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("error reading input:", err)
			return
		}
		q = strings.TrimSpace(q)
		if q == "" {
			fmt.Println("bye")
			return
		}

		// Short cooldown to avoid hammering the LLM if user spams enter.
		time.Sleep(200 * time.Millisecond)

		res, err := agent.Ask(ctx, q)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}

		fmt.Printf("\nSQL:\n%s\n\n", res.SQL)
		fmt.Printf("Answer:\n%s\n\n", res.Answer)
	}
}
