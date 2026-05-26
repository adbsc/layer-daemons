package rpchandler

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/tellor-io/layer-daemons/constants"
	reader "github.com/tellor-io/layer-daemons/custom_query/rpc/rpc_reader"
	marketParam "github.com/tellor-io/layer-daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer-daemons/server/types/pricefeed"
)

type GenericHandler struct{}

// FetchValue fetches a price from a generic REST/JSON endpoint.
//
// Optional endpoint params for data age validation:
//   - timestamp_path: dot-separated path to a timestamp field in the response (e.g. "last_updated_at").
//   - timestamp_format: "unix", "unix_ms", or a Go time layout string; defaults to RFC3339.
//
// When timestamp_path is set the parsed field is used as the data time; otherwise the
// fetch start time is used as a proxy.
func (h *GenericHandler) FetchValue(
	ctx context.Context, reader *reader.Reader, invert bool, usdViaID uint32,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
	maxDataAge time.Duration,
) (float64, error) {
	fetchedAt := time.Now()
	resp, err := reader.FetchJSON(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch JSON: %w", err)
	}

	// Determine data timestamp: use a configured field when available, else fetch time.
	dataTime := fetchedAt
	if tsPath := reader.Params["timestamp_path"]; tsPath != "" {
		var raw map[string]any
		if jsonErr := json.Unmarshal(resp, &raw); jsonErr == nil {
			if t, parseErr := parseTimestampParam(raw, tsPath, reader.Params["timestamp_format"]); parseErr == nil {
				dataTime = t
			}
		}
	}
	if err := checkDataAge(dataTime, maxDataAge); err != nil {
		return 0, fmt.Errorf("generic handler: %w", err)
	}

	current, err := reader.ExtractValueFromJSON(resp, reader.ResponsePath)
	if err != nil {
		return 0, fmt.Errorf("failed to extract value from JSON: %w", err)
	}
	var value float64
	switch v := current.(type) {
	case float64:
		value = v
	case float32:
		value = float64(v)
	case int:
		value = float64(v)
	case int64:
		value = float64(v)
	case string:
		_, err := fmt.Sscanf(v, "%f", &value)
		if err != nil {
			return 0, fmt.Errorf("error parsing string as float: %w", err)
		}
	default:
		return 0, fmt.Errorf("unsupported value type: %T", current)
	}
	// Apply inversion if needed
	if invert {
		if value == 0 {
			return 0, fmt.Errorf("cannot invert zero value")
		}
		value = 1.0 / value
	}

	// Apply USD conversion via another market if specified
	if usdViaID != 0 {
		usdViaParam, found := constants.StaticMarketParamsConfig[usdViaID]
		if !found {
			return 0, fmt.Errorf("market param not found for ID %d", usdViaID)
		}

		// Get usdVia price from cache
		usdPriceMap := priceCache.GetValidMedianPrices([]marketParam.MarketParam{*usdViaParam}, time.Now())
		usdPriceRaw, found := usdPriceMap[usdViaID]
		if !found {
			return 0, fmt.Errorf("no valid USD via price found in cache for market ID %d", usdViaID)
		}

		// Convert raw price to float using the market's exponent
		usdPrice := float64(usdPriceRaw) * math.Pow10(int(usdViaParam.Exponent))

		// Multiply the value by the USD price
		value *= usdPrice
	}

	return value, nil
}
