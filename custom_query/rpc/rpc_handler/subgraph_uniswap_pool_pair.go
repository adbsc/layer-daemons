package rpchandler

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/tellor-io/layer-daemons/constants"
	reader "github.com/tellor-io/layer-daemons/custom_query/rpc/rpc_reader"
	marketParam "github.com/tellor-io/layer-daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer-daemons/server/types/pricefeed"
)

// SubgraphUniswapPoolPairHandler reads Uniswap v3/v4-style Pool.token0Price / token1Price from a The Graph
// deployment (see theGraphUniswapStylePool in constants). Uniswap subgraph semantics match pricing.ts:
//   - token0Price = amount of token0 per 1 token1
//   - token1Price = amount of token1 per 1 token0
//
// So "quote per target" is token1Price when target is token0, and token0Price when target is token1.
//
// Reader.Params:
//   - target_token: token to express in USD
//   - quote_token: the other pool leg; usd_via_id converts this leg to USD (e.g. USDC/USD).
//
// Pool id and subgraph id belong in the endpoint URL/query template params, not here.
type SubgraphUniswapPoolPairHandler struct{}

type subgraphPoolGQLTop struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
	Data *struct {
		Pool *subgraphPoolGQL `json:"pool"`
		Meta *subgraphMeta    `json:"_meta"`
	} `json:"data"`
}

type subgraphPoolGQL struct {
	Token0      *subgraphPoolTokenRef `json:"token0"`
	Token1      *subgraphPoolTokenRef `json:"token1"`
	Token0Price string                `json:"token0Price"`
	Token1Price string                `json:"token1Price"`
}

type subgraphPoolTokenRef struct {
	ID string `json:"id"`
}

type subgraphMeta struct {
	Block struct {
		// Unix timestamp of the most recently indexed block.
		Timestamp int64 `json:"timestamp"`
	} `json:"block"`
}

func (h *SubgraphUniswapPoolPairHandler) FetchValue(
	ctx context.Context, r *reader.Reader, invert bool, usdViaID uint32,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
	maxDataAge time.Duration,
) (float64, error) {
	if usdViaID == 0 {
		return 0, fmt.Errorf("subgraph uniswap pool pair: usd_via_id is required")
	}
	if r == nil || r.Params == nil {
		return 0, fmt.Errorf("subgraph uniswap pool pair: reader params are required (target_token, quote_token)")
	}
	target := strings.ToLower(strings.TrimSpace(r.Params["target_token"]))
	quote := strings.ToLower(strings.TrimSpace(r.Params["quote_token"]))
	if target == "" || quote == "" {
		return 0, fmt.Errorf("subgraph uniswap pool pair: target_token and quote_token params are required")
	}
	if target == quote {
		return 0, fmt.Errorf("subgraph uniswap pool pair: target_token and quote_token must differ")
	}

	body, err := r.FetchJSON(ctx)
	if err != nil {
		return 0, fmt.Errorf("subgraph uniswap pool pair: fetch: %w", err)
	}

	var top subgraphPoolGQLTop
	if err := json.Unmarshal(body, &top); err != nil {
		return 0, fmt.Errorf("subgraph uniswap pool pair: json: %w", err)
	}
	if len(top.Errors) > 0 {
		return 0, fmt.Errorf("subgraph uniswap pool pair: graphql: %s", top.Errors[0].Message)
	}
	if top.Data == nil || top.Data.Pool == nil {
		return 0, fmt.Errorf("subgraph uniswap pool pair: empty data.pool (wrong pool id or subgraph)")
	}

	p := top.Data.Pool

	// Check subgraph freshness using the indexed block timestamp from _meta.
	if top.Data.Meta != nil && top.Data.Meta.Block.Timestamp > 0 {
		dataTime := time.Unix(top.Data.Meta.Block.Timestamp, 0)
		if err := checkDataAge(dataTime, maxDataAge); err != nil {
			return 0, fmt.Errorf("subgraph uniswap pool pair: %w", err)
		}
	}
	if p.Token0 == nil || p.Token1 == nil {
		return 0, fmt.Errorf("subgraph uniswap pool pair: missing token0/token1")
	}

	t0 := strings.ToLower(strings.TrimSpace(p.Token0.ID))
	t1 := strings.ToLower(strings.TrimSpace(p.Token1.ID))

	var priceInQuote float64
	switch {
	case t0 == target && t1 == quote:
		// target is token0: quote per target = token1 per token0 = token1Price
		priceInQuote, err = parseSubgraphPrice(p.Token1Price)
		if err != nil {
			return 0, fmt.Errorf("token1Price: %w", err)
		}
	case t0 == quote && t1 == target:
		// target is token1: quote per target = token0 per token1 = token0Price
		priceInQuote, err = parseSubgraphPrice(p.Token0Price)
		if err != nil {
			return 0, fmt.Errorf("token0Price: %w", err)
		}
	default:
		return 0, fmt.Errorf("subgraph uniswap pool pair: pool tokens token0=%s token1=%s do not match target=%s quote=%s",
			t0, t1, target, quote)
	}

	if priceInQuote <= 0 || math.IsNaN(priceInQuote) || math.IsInf(priceInQuote, 0) {
		return 0, fmt.Errorf("subgraph uniswap pool pair: non-positive price")
	}

	if invert {
		if priceInQuote == 0 {
			return 0, fmt.Errorf("subgraph uniswap pool pair: cannot invert zero price")
		}
		priceInQuote = 1.0 / priceInQuote
	}

	usdViaParam, found := constants.StaticMarketParamsConfig[usdViaID]
	if !found {
		return 0, fmt.Errorf("market param not found for usd_via id %d", usdViaID)
	}
	usdPriceMap := priceCache.GetValidMedianPrices([]marketParam.MarketParam{*usdViaParam}, time.Now())
	usdPriceRaw, found := usdPriceMap[usdViaID]
	if !found {
		return 0, fmt.Errorf("no valid USD via price in cache for market id %d", usdViaID)
	}
	usdPrice := float64(usdPriceRaw) * math.Pow10(int(usdViaParam.Exponent))

	return priceInQuote * usdPrice, nil
}

func parseSubgraphPrice(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty price string")
	}
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	if err != nil {
		return 0, fmt.Errorf("parse price %q: %w", s, err)
	}
	return v, nil
}
