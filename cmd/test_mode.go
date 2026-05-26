package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tellor-io/layer-daemons/configs"
	"github.com/tellor-io/layer-daemons/constants"
	customquery "github.com/tellor-io/layer-daemons/custom_query"
	"github.com/tellor-io/layer-daemons/lib"
	libtime "github.com/tellor-io/layer-daemons/lib/time"
	handler "github.com/tellor-io/layer-daemons/pricefeed/client/queryhandler"
	"github.com/tellor-io/layer-daemons/pricefeed/client/types"
	serverdaemon "github.com/tellor-io/layer-daemons/server/types/daemons"
	pricefeedservertypes "github.com/tellor-io/layer-daemons/server/types/pricefeed"
	daemontypes "github.com/tellor-io/layer-daemons/types"

	"cosmossdk.io/log"
)

// exchangeTestResult represents the result of testing an exchange for a market
type exchangeTestResult struct {
	Success bool
	Price   uint64
	Error   string
}

// runTestMode loads all price feed configurations and tests them.
// If isolatedQueryID is non-empty, only that custom query is run (no exchange/market tests).
func runTestMode(homePath string, logger log.Logger, isolatedQueryID string) error {
	logger.Info("Starting test mode - verifying price feed configurations")

	if isolatedQueryID != "" {
		logger.Info("Isolated custom query test (--test-query-id); skipping exchange/market tests")
		customQueries, err := customquery.BuildQueryEndpoints(homePath, "config", "custom_query_config.toml")
		if err != nil {
			return fmt.Errorf("load custom queries: %w", err)
		}
		id := strings.ToLower(strings.TrimSpace(isolatedQueryID))
		qc, ok := customQueries[id]
		if !ok {
			return fmt.Errorf("custom query id not found in config: %s", id)
		}
		priceCache := pricefeedservertypes.NewMarketToExchangePrices(5 * time.Minute)
		populateTestModePriceCacheFromExchanges(homePath, priceCache, logger)
		if err := testCustomQuery(id, qc, priceCache, logger); err != nil {
			return fmt.Errorf("custom query test failed: %w", err)
		}
		logger.Info("Isolated custom query test succeeded", "query_id", id)
		return nil
	}

	// Load configurations
	logger.Info("Loading market parameters...")
	marketParams := configs.ReadMarketParamsConfigFile(homePath)
	logger.Info("Loaded market parameters", "count", len(marketParams))

	logger.Info("Loading exchange configurations...")
	exchangeConfigs := configs.ReadExchangeQueryConfigFile(homePath)
	logger.Info("Loaded exchange configurations", "count", len(exchangeConfigs))

	// Load custom queries
	logger.Info("Loading custom query configurations...")
	customQueries, err := customquery.BuildQueryEndpoints(homePath, "config", "custom_query_config.toml")
	if err != nil {
		logger.Warn("Failed to load custom queries (may not exist)", "error", err)
		customQueries = make(map[string]customquery.QueryConfig)
	} else {
		logger.Info("Loaded custom query configurations", "count", len(customQueries))
	}

	// Test each market param
	logger.Info("Testing price feeds...")
	for _, marketParam := range marketParams {
		if err := testMarketParam(marketParam, exchangeConfigs, logger); err != nil {
			logger.Error("Failed to test market param", "pair", marketParam.Pair, "id", marketParam.Id, "error", err)
		}
	}

	// Test custom queries
	if len(customQueries) > 0 {
		logger.Info("Testing custom queries...")
		priceCache := pricefeedservertypes.NewMarketToExchangePrices(5 * time.Minute)
		populateTestModePriceCacheFromExchanges(homePath, priceCache, logger)
		for queryId, queryConfig := range customQueries {
			if err := testCustomQuery(queryId, queryConfig, priceCache, logger); err != nil {
				logger.Error("Failed to test custom query", "query_id", queryId, "error", err)
			}
		}
	}

	logger.Info("Test mode completed successfully")
	return nil
}

// testMarketParam tests a single market param by querying all configured exchanges
func testMarketParam(
	marketParam types.MarketParam,
	exchangeConfigs map[types.ExchangeId]*types.ExchangeQueryConfig,
	logger log.Logger,
) error {
	logger.Info("Testing market", "pair", marketParam.Pair, "id", marketParam.Id)

	// Parse ExchangeConfigJson to get exchange ticker mappings
	var exchangeConfigJson types.ExchangeConfigJson
	if err := json.Unmarshal([]byte(marketParam.ExchangeConfigJson), &exchangeConfigJson); err != nil {
		return fmt.Errorf("failed to parse exchange config JSON: %w", err)
	}

	// Build market name to ID mapping for adjust-by-market support
	// For test mode, we'll skip adjust-by-market validation as it requires all market params
	marketNameToId := make(map[string]types.MarketId)
	marketNameToId[marketParam.Pair] = marketParam.Id

	// Track results for each exchange
	exchangeResults := make(map[types.ExchangeId]exchangeTestResult)
	var validPrices []uint64

	// Test each configured exchange
	for _, exchangeConfigJsonItem := range exchangeConfigJson.Exchanges {
		exchangeId := exchangeConfigJsonItem.ExchangeName

		// Check if exchange details exist
		exchangeDetails, exists := constants.StaticExchangeDetails[exchangeId]
		if !exists {
			exchangeResults[exchangeId] = exchangeTestResult{
				Success: false,
				Error:   "no exchange details found",
			}
			continue
		}

		// Check if exchange query config exists
		exchangeQueryConfig, hasConfig := exchangeConfigs[exchangeId]
		if !hasConfig {
			exchangeResults[exchangeId] = exchangeTestResult{
				Success: false,
				Error:   "exchange query config not found",
			}
			continue
		}

		// Query the exchange
		result := queryExchangeForMarket(
			exchangeId,
			exchangeDetails,
			*exchangeQueryConfig,
			marketParam,
			exchangeConfigJsonItem,
			logger,
		)
		exchangeResults[exchangeId] = result

		if result.Success {
			validPrices = append(validPrices, result.Price)
		}
	}

	// Calculate median
	var medianPrice uint64
	var medianErr error
	if len(validPrices) >= int(marketParam.MinExchanges) {
		medianPrice, medianErr = lib.Median[uint64](validPrices)
	} else {
		medianErr = fmt.Errorf("insufficient valid prices: got %d, need %d", len(validPrices), marketParam.MinExchanges)
	}

	// Log results
	logger.Info("Market test results",
		"pair", marketParam.Pair,
		"id", marketParam.Id,
		"valid_sources", len(validPrices),
		"total_sources", len(exchangeResults),
		"min_required", marketParam.MinExchanges,
		"median_price", medianPrice,
		"median_error", medianErr,
	)

	// Log individual exchange results
	for exchangeId, result := range exchangeResults {
		if result.Success {
			logger.Info("  ✓ Exchange succeeded",
				"exchange", exchangeId,
				"price", result.Price,
			)
		} else {
			logger.Warn("  ✗ Exchange failed",
				"exchange", exchangeId,
				"error", result.Error,
			)
		}
	}

	if medianErr != nil {
		return fmt.Errorf("failed to calculate median: %w", medianErr)
	}

	return nil
}

// queryExchangeForMarket queries a single exchange for a market and returns the result
func queryExchangeForMarket(
	exchangeId types.ExchangeId,
	exchangeDetails types.ExchangeQueryDetails,
	exchangeConfig types.ExchangeQueryConfig,
	marketParam types.MarketParam,
	exchangeConfigJsonItem types.ExchangeMarketConfigJson,
	logger log.Logger,
) exchangeTestResult {
	// Create a mutable exchange market config for this test
	mutableConfig := &types.MutableExchangeMarketConfig{
		Id:                   exchangeId,
		MarketToMarketConfig: make(map[types.MarketId]types.MarketConfig),
	}

	// Create market config from exchange config JSON
	marketConfig := types.MarketConfig{
		Ticker: exchangeConfigJsonItem.Ticker,
		Invert: exchangeConfigJsonItem.Invert,
	}

	// Handle AdjustByMarket if present (for test mode, we'll skip this as it requires
	// all market params to be loaded, which is complex for a simple test)
	if exchangeConfigJsonItem.AdjustByMarket != "" {
		// In a full implementation, we'd look up the market ID here
		// For test mode, we'll log a warning and continue
		logger.Debug("AdjustByMarket specified but not fully supported in test mode",
			"exchange", exchangeId,
			"adjust_by_market", exchangeConfigJsonItem.AdjustByMarket,
		)
	}

	mutableConfig.MarketToMarketConfig[marketParam.Id] = marketConfig

	// Create market price exponent map
	marketExponents := map[types.MarketId]types.Exponent{
		marketParam.Id: marketParam.Exponent,
	}

	// Create query handler
	queryHandler := &handler.ExchangeQueryHandlerImpl{
		TimeProvider: &libtime.TimeProviderImpl{},
	}

	// Create request handler with a new HTTP client
	httpClient := &http.Client{
		Timeout: time.Duration(exchangeConfig.TimeoutMs) * time.Millisecond,
	}
	requestHandler := daemontypes.NewRequestHandlerImpl(httpClient)

	// Query with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(exchangeConfig.TimeoutMs)*time.Millisecond)
	defer cancel()

	marketIds := []types.MarketId{marketParam.Id}

	prices, unavailableMarkets, err := queryHandler.Query(
		ctx,
		&exchangeDetails,
		mutableConfig,
		marketIds,
		requestHandler,
		marketExponents,
	)
	if err != nil {
		return exchangeTestResult{
			Success: false,
			Error:   err.Error(),
		}
	}

	if len(unavailableMarkets) > 0 {
		if err, ok := unavailableMarkets[marketParam.Id]; ok {
			return exchangeTestResult{
				Success: false,
				Error:   fmt.Sprintf("market unavailable: %v", err),
			}
		}
	}

	if len(prices) == 0 {
		return exchangeTestResult{
			Success: false,
			Error:   "no prices returned",
		}
	}

	return exchangeTestResult{
		Success: true,
		Price:   prices[0].Price,
	}
}

// testCustomQuery tests a single custom query configuration.
// priceCache must already be populated (e.g. populateTestModePriceCacheFromExchanges) so handlers
// that resolve USD via other markets use live exchange medians instead of synthetic values.
func testCustomQuery(queryId string, queryConfig customquery.QueryConfig, priceCache *pricefeedservertypes.MarketToExchangePrices, logger log.Logger) error {
	logger.Info("Testing custom query", "query_id", queryId)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := customquery.FetchPrice(ctx, queryConfig, priceCache)
	logCustomQuerySourceResults(logger, queryId, results)
	if err != nil {
		logger.Warn("  ✗ Custom query failed",
			"query_id", queryId,
			"error", err,
		)
		return err
	}

	logger.Info("  ✓ Custom query succeeded",
		"query_id", queryId,
		"encoded_value", results.EncodedValue,
		"success_rate", results.SuccessRate,
	)

	return nil
}

// logCustomQuerySourceResults prints one line per configured endpoint (contract, RPC, or combined).
// On failure, FetchPrice may still return results with RawResults populated for diagnostics.
func logCustomQuerySourceResults(logger log.Logger, queryId string, fr *customquery.FetchPriceResult) {
	if fr == nil || len(fr.RawResults) == 0 {
		logger.Info("custom query per-source results", "query_id", queryId, "count", 0, "note", "no per-endpoint results")
		return
	}
	for i, r := range fr.RawResults {
		if r.Err != nil {
			logger.Info("custom query per-source result",
				"query_id", queryId,
				"source_index", i,
				"source_id", r.SourceId,
				"endpoint_id", r.EndpointID,
				"market_id", r.MarketId,
				"ok", false,
				"error", r.Err.Error(),
			)
			continue
		}
		logger.Info("custom query per-source result",
			"query_id", queryId,
			"source_index", i,
			"source_id", r.SourceId,
			"endpoint_id", r.EndpointID,
			"market_id", r.MarketId,
			"ok", true,
			"price", r.Value,
		)
	}
}

// populateTestModePriceCacheFromExchanges fills the cache with the same per-exchange prices used
// in market-param tests, so custom-query handlers that call GetValidMedianPrices see real sources.
func populateTestModePriceCacheFromExchanges(
	homePath string,
	cache *pricefeedservertypes.MarketToExchangePrices,
	logger log.Logger,
) {
	marketParams := configs.ReadMarketParamsConfigFile(homePath)
	exchangeConfigs := configs.ReadExchangeQueryConfigFile(homePath)
	var updates []*serverdaemon.MarketPriceUpdate
	for i := range marketParams {
		u := marketPriceUpdateFromLiveExchanges(&marketParams[i], exchangeConfigs, logger)
		if u != nil {
			updates = append(updates, u)
		}
	}
	if len(updates) > 0 {
		cache.UpdatePrices(updates)
	}
	if len(updates) == 0 {
		logger.Warn("No markets had enough live exchange prices to populate --test reference cache; custom queries that need USD-via may fail")
	} else {
		logger.Info("Populated --test price reference cache from live exchanges",
			"markets_with_sufficient_exchanges", len(updates),
			"total_market_params", len(marketParams),
		)
	}
}

// marketPriceUpdateFromLiveExchanges returns a single-market update when at least MinExchanges
// configured sources succeed; otherwise nil.
func marketPriceUpdateFromLiveExchanges(
	marketParam *types.MarketParam,
	exchangeConfigs map[types.ExchangeId]*types.ExchangeQueryConfig,
	logger log.Logger,
) *serverdaemon.MarketPriceUpdate {
	var exchangeConfigJson types.ExchangeConfigJson
	if err := json.Unmarshal([]byte(marketParam.ExchangeConfigJson), &exchangeConfigJson); err != nil {
		logger.Debug("Skipping market for test cache: invalid exchange config JSON",
			"pair", marketParam.Pair, "error", err)
		return nil
	}
	now := time.Now()
	var exchangePrices []*serverdaemon.ExchangePrice
	for _, exchangeConfigJsonItem := range exchangeConfigJson.Exchanges {
		exchangeId := exchangeConfigJsonItem.ExchangeName
		exchangeDetails, exists := constants.StaticExchangeDetails[exchangeId]
		if !exists {
			continue
		}
		exchangeQueryConfig, hasConfig := exchangeConfigs[exchangeId]
		if !hasConfig {
			continue
		}
		result := queryExchangeForMarket(
			exchangeId,
			exchangeDetails,
			*exchangeQueryConfig,
			*marketParam,
			exchangeConfigJsonItem,
			logger,
		)
		if !result.Success {
			continue
		}
		t := now
		exchangePrices = append(exchangePrices, &serverdaemon.ExchangePrice{
			ExchangeId:     exchangeId,
			Price:          result.Price,
			LastUpdateTime: &t,
		})
	}
	if len(exchangePrices) < int(marketParam.MinExchanges) {
		logger.Debug("Skipping market for test cache: insufficient live exchange prices",
			"pair", marketParam.Pair,
			"market_id", marketParam.Id,
			"have", len(exchangePrices),
			"need", marketParam.MinExchanges,
		)
		return nil
	}
	return &serverdaemon.MarketPriceUpdate{
		MarketId:       marketParam.Id,
		ExchangePrices: exchangePrices,
	}
}
