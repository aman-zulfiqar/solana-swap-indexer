package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aman-zulfiqar/solana-swap-indexer/internal/jupiter"
	"github.com/labstack/echo/v4"
)

func splitCSVQuery(values []string) []string {
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		parts := strings.Split(v, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

func (h *Handlers) Quote(c echo.Context) error {
	if h.Jupiter == nil {
		return h.err(c, http.StatusBadRequest, "jupiter is not configured", nil)
	}

	inputMint := strings.TrimSpace(c.QueryParam("inputMint"))
	outputMint := strings.TrimSpace(c.QueryParam("outputMint"))
	amountStr := strings.TrimSpace(c.QueryParam("amount"))

	if inputMint == "" {
		return h.err(c, http.StatusBadRequest, "invalid inputMint", map[string]any{"inputMint": "required"})
	}
	if outputMint == "" {
		return h.err(c, http.StatusBadRequest, "invalid outputMint", map[string]any{"outputMint": "required"})
	}
	if amountStr == "" {
		return h.err(c, http.StatusBadRequest, "invalid amount", map[string]any{"amount": "required"})
	}
	if _, err := strconv.ParseUint(amountStr, 10, 64); err != nil {
		return h.err(c, http.StatusBadRequest, "invalid amount", map[string]any{"amount": "must be uint64"})
	}

	var slippageBps *uint16
	if v := strings.TrimSpace(c.QueryParam("slippageBps")); v != "" {
		n, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return h.err(c, http.StatusBadRequest, "invalid slippageBps", map[string]any{"slippageBps": "must be uint16"})
		}
		tmp := uint16(n)
		slippageBps = &tmp
	}

	swapMode := strings.TrimSpace(c.QueryParam("swapMode"))
	if swapMode != "" && swapMode != "ExactIn" && swapMode != "ExactOut" {
		return h.err(c, http.StatusBadRequest, "invalid swapMode", map[string]any{"swapMode": "must be ExactIn or ExactOut"})
	}

	var restrictIntermediateTokens *bool
	if v := strings.TrimSpace(c.QueryParam("restrictIntermediateTokens")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return h.err(c, http.StatusBadRequest, "invalid restrictIntermediateTokens", map[string]any{"restrictIntermediateTokens": "must be boolean"})
		}
		restrictIntermediateTokens = &b
	}

	var onlyDirectRoutes *bool
	if v := strings.TrimSpace(c.QueryParam("onlyDirectRoutes")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return h.err(c, http.StatusBadRequest, "invalid onlyDirectRoutes", map[string]any{"onlyDirectRoutes": "must be boolean"})
		}
		onlyDirectRoutes = &b
	}

	var asLegacyTransaction *bool
	if v := strings.TrimSpace(c.QueryParam("asLegacyTransaction")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return h.err(c, http.StatusBadRequest, "invalid asLegacyTransaction", map[string]any{"asLegacyTransaction": "must be boolean"})
		}
		asLegacyTransaction = &b
	}

	var platformFeeBps *uint16
	if v := strings.TrimSpace(c.QueryParam("platformFeeBps")); v != "" {
		n, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return h.err(c, http.StatusBadRequest, "invalid platformFeeBps", map[string]any{"platformFeeBps": "must be uint16"})
		}
		tmp := uint16(n)
		platformFeeBps = &tmp
	}

	var maxAccounts *uint64
	if v := strings.TrimSpace(c.QueryParam("maxAccounts")); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return h.err(c, http.StatusBadRequest, "invalid maxAccounts", map[string]any{"maxAccounts": "must be uint64"})
		}
		maxAccounts = &n
	}

	instructionVersion := strings.TrimSpace(c.QueryParam("instructionVersion"))
	if instructionVersion != "" && instructionVersion != "V1" && instructionVersion != "V2" {
		return h.err(c, http.StatusBadRequest, "invalid instructionVersion", map[string]any{"instructionVersion": "must be V1 or V2"})
	}

	var dynamicSlippage *bool
	if v := strings.TrimSpace(c.QueryParam("dynamicSlippage")); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return h.err(c, http.StatusBadRequest, "invalid dynamicSlippage", map[string]any{"dynamicSlippage": "must be boolean"})
		}
		dynamicSlippage = &b
	}

	dexes := splitCSVQuery(c.QueryParams()["dexes"])
	excludeDexes := splitCSVQuery(c.QueryParams()["excludeDexes"])

	ctx, cancel := h.withTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	out, err := h.Jupiter.Quote(ctx, jupiter.QuoteRequest{
		InputMint:                  inputMint,
		OutputMint:                 outputMint,
		Amount:                     amountStr,
		SlippageBps:                slippageBps,
		SwapMode:                   swapMode,
		Dexes:                      dexes,
		ExcludeDexes:               excludeDexes,
		RestrictIntermediateTokens: restrictIntermediateTokens,
		OnlyDirectRoutes:           onlyDirectRoutes,
		AsLegacyTransaction:        asLegacyTransaction,
		PlatformFeeBps:             platformFeeBps,
		MaxAccounts:                maxAccounts,
		InstructionVersion:         instructionVersion,
		DynamicSlippage:            dynamicSlippage,
	})
	if err != nil {
		return h.err(c, http.StatusBadGateway, "jupiter quote failed", map[string]any{"err": err.Error()})
	}

	return c.JSON(http.StatusOK, out)
}
