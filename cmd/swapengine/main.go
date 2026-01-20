package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/swapengine"
	"github.com/joho/godotenv"
)

func loadEnv() {
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "../..")
	_ = godotenv.Load(filepath.Join(projectRoot, ".env"))
}

func main() {
	loadEnv()

	mode := flag.String("mode", "quote", "quote | execute")
	inTok := flag.String("in", "SOL", "input token symbol (e.g. SOL)")
	outTok := flag.String("out", "USDC", "output token symbol (e.g. USDC)")
	amt := flag.Float64("amt", 0, "amount in human units (e.g. 0.1)")
	slippageBps := flag.Int("slippage-bps", 100, "slippage in bps (e.g. 100 = 1%)")
	flag.Parse()

	if *amt <= 0 {
		fmt.Println("missing -amt (must be > 0)")
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	engine, err := swapengine.NewEngineFromEnv()
	if err != nil {
		fmt.Println("failed to init swapengine:", err)
		os.Exit(1)
	}
	defer engine.Close()

	slip := uint16(*slippageBps)
	intent := &swapengine.SwapIntent{
		InputToken:  *inTok,
		OutputToken: *outTok,
		Amount:      *amt,
		SlippageBps: &slip,
		RequestedAt: time.Now(),
	}

	switch *mode {
	case "quote":
		q, err := engine.GetQuote(ctx, intent)
		if err != nil {
			fmt.Println("quote failed:", err)
			os.Exit(1)
		}
		fmt.Printf("pool=%s amount_in=%d amount_out=%d min_out=%d price_impact=%.4f fee_bps=%d\n",
			q.PoolName, q.AmountIn, q.AmountOut, q.MinAmountOut, q.PriceImpact, q.FeeBps)
	case "execute":
		res, err := engine.ExecuteAISwap(ctx, intent)
		if err != nil {
			fmt.Println("execute failed:", err)
			os.Exit(1)
		}
		fmt.Printf("success=%v sig=%s duration=%s\n", res.Success, res.Signature, res.Duration)
	default:
		fmt.Println("invalid -mode (use quote|execute)")
		os.Exit(2)
	}
}

