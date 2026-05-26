package rpchandler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/tellor-io/layer-daemons/constants"
	reader "github.com/tellor-io/layer-daemons/custom_query/rpc/rpc_reader"
	marketParam "github.com/tellor-io/layer-daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer-daemons/server/types/pricefeed"
)

type OsmosisPoolPriceHandler struct{}

const (
	STATOM_ADDRESS = "ibc/C140AFD542AE77BD7DCC83F13FDD8C5E5BB8C4929785E6EC2F4C636F98F17901"
	ATOM_ADDRESS   = "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
)

func (h *OsmosisPoolPriceHandler) FetchValue(
	ctx context.Context, reader *reader.Reader, invert bool, usdViaID uint32,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
	maxDataAge time.Duration,
) (float64, error) {
	resp, err := reader.FetchJSON(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch JSON: %w", err)
	}
	value, err := reader.ExtractValueFromJSON(resp, reader.ResponsePath)
	if err != nil {
		return 0, fmt.Errorf("failed to extract value from JSON: %w", err)
	}
	if value == nil {
		return 0, fmt.Errorf("no value found at response path %v", reader.ResponsePath)
	}
	data, ok := value.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("expected a dictionary for value, got %T", value)
	}

	// Use last_liquidity_update as the authoritative data timestamp when present.
	if rawTs, exists := data["last_liquidity_update"]; exists {
		if tsStr, ok := rawTs.(string); ok && tsStr != "" {
			dataTime, err := time.Parse(time.RFC3339Nano, tsStr)
			if err != nil {
				dataTime, err = time.Parse(time.RFC3339, tsStr)
			}
			if err == nil {
				if err := checkDataAge(dataTime, maxDataAge); err != nil {
					return 0, fmt.Errorf("osmosis pool: %w", err)
				}
			}
		}
	}

	targetToken, quoteToken, err := osmosisPairParams(reader)
	if err != nil {
		return 0, err
	}

	currentPrice, err := osmosisPriceInQuote(data, targetToken, quoteToken, reader.Params)
	if err != nil {
		return 0, err
	}
	if invert {
		if currentPrice == 0 {
			return 0, errors.New("cannot invert zero price")
		}
		currentPrice = 1 / currentPrice
	}

	usdViaParam, found := constants.StaticMarketParamsConfig[usdViaID]
	if !found {
		return 0, fmt.Errorf("market param not found for ID %d", usdViaID)
	}

	usdPriceMap := priceCache.GetValidMedianPrices([]marketParam.MarketParam{*usdViaParam}, time.Now())
	usdPriceRaw, found := usdPriceMap[usdViaID]
	if !found {
		return 0, errors.New("no valid USD via price found in cache")
	}

	usdPrice := float64(usdPriceRaw) * math.Pow10(int(usdViaParam.Exponent))
	return usdPrice * currentPrice, nil
}

func osmosisPairParams(reader *reader.Reader) (string, string, error) {
	targetToken := strings.TrimSpace(reader.Params["target_token"])
	quoteToken := strings.TrimSpace(reader.Params["quote_token"])
	if targetToken == "" && quoteToken == "" {
		// Preserve the original stATOM/ATOM configuration, which predated generic params.
		return STATOM_ADDRESS, ATOM_ADDRESS, nil
	}
	if targetToken == "" || quoteToken == "" {
		return "", "", errors.New("osmosis pool requires both target_token and quote_token params")
	}
	if strings.EqualFold(targetToken, quoteToken) {
		return "", "", errors.New("osmosis pool target_token and quote_token must differ")
	}
	return targetToken, quoteToken, nil
}

func osmosisPriceInQuote(data map[string]any, targetToken, quoteToken string, params map[string]string) (float64, error) {
	if _, ok := data["current_sqrt_price"]; ok {
		return osmosisConcentratedLiquidityPrice(data, targetToken, quoteToken)
	}
	if _, ok := data["pool_assets"]; ok {
		return osmosisWeightedPoolPrice(data, targetToken, quoteToken, params)
	}
	return 0, errors.New("unsupported osmosis pool type")
}

func osmosisConcentratedLiquidityPrice(data map[string]any, targetToken, quoteToken string) (float64, error) {
	currentSqrtPrice, ok := data["current_sqrt_price"]
	if !ok {
		return 0, fmt.Errorf("current_sqrt_price not found in JSON")
	}
	token0, ok := data["token0"].(string)
	if !ok {
		return 0, fmt.Errorf("token0 not found in JSON")
	}
	token1, ok := data["token1"].(string)
	if !ok {
		return 0, fmt.Errorf("token1 not found in JSON")
	}
	var sqrtPrice float64
	switch v := currentSqrtPrice.(type) {
	case float64:
		sqrtPrice = v
	case string:
		var err error
		sqrtPrice, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse sqrt price as float: %w", err)
		}
	default:
		return 0, fmt.Errorf("unexpected type for sqrt price: %T", currentSqrtPrice)
	}

	price := sqrtPrice * sqrtPrice
	switch {
	case strings.EqualFold(token0, targetToken) && strings.EqualFold(token1, quoteToken):
		return price, nil
	case strings.EqualFold(token0, quoteToken) && strings.EqualFold(token1, targetToken):
		if price == 0 {
			return 0, errors.New("cannot invert zero osmosis concentrated liquidity price")
		}
		return 1 / price, nil
	default:
		return 0, fmt.Errorf("osmosis pool tokens token0=%s token1=%s do not match target=%s quote=%s",
			token0, token1, targetToken, quoteToken)
	}
}

func osmosisWeightedPoolPrice(data map[string]any, targetToken, quoteToken string, params map[string]string) (float64, error) {
	assets, ok := data["pool_assets"].([]any)
	if !ok {
		return 0, fmt.Errorf("pool_assets not found in JSON")
	}

	target, foundTarget, err := osmosisWeightedPoolAsset(assets, targetToken)
	if err != nil {
		return 0, err
	}
	quote, foundQuote, err := osmosisWeightedPoolAsset(assets, quoteToken)
	if err != nil {
		return 0, err
	}
	if !foundTarget || !foundQuote {
		return 0, fmt.Errorf("osmosis weighted pool does not contain target=%s and quote=%s", targetToken, quoteToken)
	}

	targetDecimals, err := osmosisOptionalDecimals(params, "target_decimals")
	if err != nil {
		return 0, err
	}
	quoteDecimals, err := osmosisOptionalDecimals(params, "quote_decimals")
	if err != nil {
		return 0, err
	}

	targetAmount := target.amount / math.Pow10(targetDecimals)
	quoteAmount := quote.amount / math.Pow10(quoteDecimals)
	if targetAmount <= 0 || quoteAmount <= 0 || target.weight <= 0 || quote.weight <= 0 {
		return 0, errors.New("osmosis weighted pool has non-positive amount or weight")
	}

	return (quoteAmount / quote.weight) / (targetAmount / target.weight), nil
}

type osmosisWeightedAsset struct {
	amount float64
	weight float64
}

func osmosisWeightedPoolAsset(assets []any, denom string) (osmosisWeightedAsset, bool, error) {
	for _, rawAsset := range assets {
		asset, ok := rawAsset.(map[string]any)
		if !ok {
			return osmosisWeightedAsset{}, false, fmt.Errorf("expected pool asset object, got %T", rawAsset)
		}
		token, ok := asset["token"].(map[string]any)
		if !ok {
			return osmosisWeightedAsset{}, false, errors.New("pool asset missing token object")
		}
		tokenDenom, ok := token["denom"].(string)
		if !ok {
			return osmosisWeightedAsset{}, false, errors.New("pool asset token missing denom")
		}
		if !strings.EqualFold(tokenDenom, denom) {
			continue
		}

		amount, err := osmosisDecimalString(token["amount"])
		if err != nil {
			return osmosisWeightedAsset{}, false, fmt.Errorf("pool asset amount: %w", err)
		}
		weight, err := osmosisDecimalString(asset["weight"])
		if err != nil {
			return osmosisWeightedAsset{}, false, fmt.Errorf("pool asset weight: %w", err)
		}
		return osmosisWeightedAsset{amount: amount, weight: weight}, true, nil
	}
	return osmosisWeightedAsset{}, false, nil
}

func osmosisDecimalString(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case string:
		f, ok := new(big.Float).SetString(strings.TrimSpace(v))
		if !ok {
			return 0, fmt.Errorf("failed to parse decimal %q", v)
		}
		out, _ := f.Float64()
		return out, nil
	default:
		return 0, fmt.Errorf("unexpected decimal type %T", value)
	}
}

func osmosisOptionalDecimals(params map[string]string, key string) (int, error) {
	if params == nil || strings.TrimSpace(params[key]) == "" {
		return 0, nil
	}
	decimals, err := strconv.Atoi(strings.TrimSpace(params[key]))
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return decimals, nil
}
