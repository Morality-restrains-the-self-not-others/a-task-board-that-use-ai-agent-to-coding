package main

import (
	"errors"
	"math/big"
	"strings"
)

var (
	errBudgetDisabled       = errors.New("llm_budget_disabled")
	errBudgetProviderInelig = errors.New("llm_budget_provider_not_sub_token")
	errBudgetTaskNotFound   = errors.New("task_not_found")
	errBudgetCompanyMissing = errors.New("todo company_id unresolved")
)

type budgetUsageRow struct {
	ID             string
	TodoID         string
	WorkspaceID    string
	CompanyID      string
	Provider       string
	BaseURL        string
	ModelName      string
	InputTokens    int64
	OutputTokens   int64
	SpentAmount    string
	LastReportedAt *string
}

type budgetRecordItem struct {
	Provider          string
	BaseURL           string
	ModelName         string
	InputTokensDelta  int64
	OutputTokensDelta int64
	IdempotencyKey    string
}

type budgetRecordResult struct {
	Provider    string `json:"provider"`
	ModelName   string `json:"model_name"`
	SpentAmount string `json:"spent_amount"`
	Created     bool   `json:"created"`
	UsageID     string `json:"usage_id,omitempty"`
}

func normalizeBaseURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/")
}

func calculateSpentCNY(inputTokens, outputTokens int64, inputPricePer1M, outputPricePer1M string) (string, error) {
	inp := inputTokens
	if inp < 0 {
		inp = 0
	}
	out := outputTokens
	if out < 0 {
		out = 0
	}
	inPrice, ok := new(big.Rat).SetString(strings.TrimSpace(inputPricePer1M))
	if !ok {
		inPrice = big.NewRat(0, 1)
	}
	outPrice, ok := new(big.Rat).SetString(strings.TrimSpace(outputPricePer1M))
	if !ok {
		outPrice = big.NewRat(0, 1)
	}
	million := big.NewRat(1000000, 1)
	inPart := new(big.Rat).Quo(new(big.Rat).SetInt64(inp), million)
	inPart.Mul(inPart, inPrice)
	outPart := new(big.Rat).Quo(new(big.Rat).SetInt64(out), million)
	outPart.Mul(outPart, outPrice)
	spent := new(big.Rat).Add(inPart, outPart)
	return quantizeRatHalfUp(spent, 6), nil
}

func quantizeRatHalfUp(r *big.Rat, places int) string {
	if r == nil {
		return "0." + strings.Repeat("0", places)
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)
	scaled := new(big.Rat).Mul(r, new(big.Rat).SetInt(scale))
	num := new(big.Int).Set(scaled.Num())
	den := new(big.Int).Set(scaled.Denom())
	quot, rem := new(big.Int).QuoRem(num, den, new(big.Int))
	twiceRem := new(big.Int).Abs(new(big.Int).Mul(rem, big.NewInt(2)))
	if twiceRem.Cmp(den) >= 0 {
		if num.Sign() >= 0 {
			quot.Add(quot, big.NewInt(1))
		} else {
			quot.Sub(quot, big.NewInt(1))
		}
	}
	neg := quot.Sign() < 0
	abs := new(big.Int).Abs(quot)
	s := abs.String()
	if len(s) <= places {
		s = strings.Repeat("0", places-len(s)+1) + s
	}
	intPart := s[:len(s)-places]
	fracPart := s[len(s)-places:]
	out := intPart + "." + fracPart
	if neg {
		out = "-" + out
	}
	return out
}

func decimalCmp(a, b string) int {
	ra, okA := new(big.Rat).SetString(strings.TrimSpace(a))
	rb, okB := new(big.Rat).SetString(strings.TrimSpace(b))
	if !okA {
		ra = big.NewRat(0, 1)
	}
	if !okB {
		rb = big.NewRat(0, 1)
	}
	return ra.Cmp(rb)
}
