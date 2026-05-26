package rpchandler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	reader "github.com/tellor-io/layer-daemons/custom_query/rpc/rpc_reader"
	pricefeedservertypes "github.com/tellor-io/layer-daemons/server/types/pricefeed"
)

// curveFactoryHTTPGet performs GETs for optional merge / v1 Curve API calls. Tests may replace.
var curveFactoryHTTPGet = curveFactoryHTTPGetDefault

func curveFactoryHTTPGetDefault(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

type curvePoolsEnvelope struct {
	Success bool `json:"success"`
	Data    *struct {
		PoolData []curvePool `json:"poolData"`
	} `json:"data"`
}

type curvePool struct {
	Address string      `json:"address"`
	Coins   []curveCoin `json:"coins"`
}

type curveCoin struct {
	Address  string  `json:"address"`
	USDPrice float64 `json:"usdPrice"`
	Symbol   string  `json:"symbol"`
}

type curveV1USD struct {
	Data *struct {
		UsdPrice float64 `json:"usd_price"`
		Address  string  `json:"address"`
	} `json:"data"`
}

// CurveFactoryPriceHandler medians Curve getPools per-coin usdPrice for a configured target token
// (same idea as telliot CurveFinanceSpotPriceService + CurveFiUSDPriceService v1 fallback).
//
// Reader.Params (from endpoint config):
//   - target_token (required): ERC-20 contract hex; matched case-insensitively against pool coins.
//   - exclude_pools (optional): comma-separated pool contract addresses to skip (e.g. bad peg pools).
//   - merge_get_pools_url (optional): extra getPools JSON URL to merge into samples (e.g. ethereum/main).
//   - v1_usd_price_url (optional): full URL for v1 usd_price fallback; default https://prices.curve.finance/v1/usd_price/ethereum/{target_token}.
type CurveFactoryPriceHandler struct{}

func (h *CurveFactoryPriceHandler) FetchValue(
	ctx context.Context, r *reader.Reader, _ bool, _ uint32,
	_ *pricefeedservertypes.MarketToExchangePrices,
	maxDataAge time.Duration,
) (float64, error) {
	if r == nil || r.Params == nil {
		return 0, fmt.Errorf("curve factory: reader params are required (target_token)")
	}
	target := strings.ToLower(strings.TrimSpace(r.Params["target_token"]))
	if target == "" {
		return 0, fmt.Errorf("curve factory: target_token param is required")
	}
	exclude := parseExcludedPoolSet(r.Params["exclude_pools"])
	mergeURL := strings.TrimSpace(r.Params["merge_get_pools_url"])

	fetchedAt := time.Now()
	body, err := r.FetchJSON(ctx)
	if err != nil {
		return 0, fmt.Errorf("curve factory: primary fetch: %w", err)
	}
	// Curve API responses carry no embedded timestamp; use fetch time as proxy.
	if err := checkDataAge(fetchedAt, maxDataAge); err != nil {
		return 0, fmt.Errorf("curve factory: %w", err)
	}

	samples, perr := collectCurveUSDPricesFromGetPoolsJSON(body, target, exclude)
	if perr != nil {
		return 0, fmt.Errorf("curve factory: primary json: %w", perr)
	}

	if mergeURL != "" {
		if extra, err := curveFactoryHTTPGet(ctx, mergeURL); err == nil {
			more, _ := collectCurveUSDPricesFromGetPoolsJSON(extra, target, exclude)
			samples = append(samples, more...)
		}
	}

	if len(samples) == 0 {
		return curveFactoryPriceFromV1USD(ctx, r.Params, target)
	}

	return medianFloat64(samples), nil
}

func parseExcludedPoolSet(s string) map[string]bool {
	out := make(map[string]bool)
	for _, part := range strings.Split(s, ",") {
		p := strings.ToLower(strings.TrimSpace(part))
		if p != "" {
			out[p] = true
		}
	}
	return out
}

func collectCurveUSDPricesFromGetPoolsJSON(body []byte, targetAddrLower string, excludePools map[string]bool) ([]float64, error) {
	var env curvePoolsEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	if !env.Success || env.Data == nil {
		return nil, fmt.Errorf("success=false or missing data")
	}

	var out []float64
	for _, pool := range env.Data.PoolData {
		poolAddr := strings.ToLower(strings.TrimSpace(pool.Address))
		if excludePools[poolAddr] {
			continue
		}
		for i := range pool.Coins {
			a := strings.ToLower(strings.TrimSpace(pool.Coins[i].Address))
			if a != targetAddrLower {
				continue
			}
			p := pool.Coins[i].USDPrice
			if p <= 0 || math.IsNaN(p) || math.IsInf(p, 0) {
				continue
			}
			out = append(out, p)
			break
		}
	}
	return out, nil
}

func curveFactoryV1URL(params map[string]string, targetLower string) string {
	if u := strings.TrimSpace(params["v1_usd_price_url"]); u != "" {
		return u
	}
	return "https://prices.curve.finance/v1/usd_price/ethereum/" + targetLower
}

func curveFactoryPriceFromV1USD(ctx context.Context, params map[string]string, targetAddrLower string) (float64, error) {
	url := curveFactoryV1URL(params, targetAddrLower)
	body, err := curveFactoryHTTPGet(ctx, url)
	if err != nil {
		return 0, fmt.Errorf("curve factory: no getPools samples and v1 usd_price fetch failed: %w", err)
	}
	var v1 curveV1USD
	if err := json.Unmarshal(body, &v1); err != nil {
		return 0, fmt.Errorf("curve factory: v1 json: %w", err)
	}
	if v1.Data == nil {
		return 0, fmt.Errorf("curve factory: v1 missing data")
	}
	if strings.ToLower(strings.TrimSpace(v1.Data.Address)) != targetAddrLower {
		return 0, fmt.Errorf("curve factory: v1 address mismatch")
	}
	if v1.Data.UsdPrice <= 0 || math.IsNaN(v1.Data.UsdPrice) {
		return 0, fmt.Errorf("curve factory: v1 invalid usd_price")
	}
	return v1.Data.UsdPrice, nil
}

func medianFloat64(samples []float64) float64 {
	s := append([]float64(nil), samples...)
	sort.Float64s(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid]
	}
	return (s[mid-1] + s[mid]) / 2
}
