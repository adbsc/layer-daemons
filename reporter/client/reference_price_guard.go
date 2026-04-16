package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"

	"cosmossdk.io/log"

	"github.com/tellor-io/layer/utils"
)

// ReferencePriceSource provides the external reference price for a query.
// Implementations can fetch from any backend (API, DB, in-memory cache, etc.).
type ReferencePriceSource interface {
	GetReferencePrice(ctx context.Context, queryData []byte) (float64, error)
}

// ReferencePriceGuard validates reporter prices against an external reference.
// It is disabled unless both a source and positive maxDeviation are provided.
type ReferencePriceGuard struct {
	source       ReferencePriceSource
	maxDeviation float64
	enabled      bool
	logger       log.Logger
}

func NewReferencePriceGuard(source ReferencePriceSource, maxDeviation float64, logger log.Logger) *ReferencePriceGuard {
	return &ReferencePriceGuard{
		source:       source,
		maxDeviation: maxDeviation,
		enabled:      source != nil && maxDeviation > 0,
		logger:       logger.With("component", "reference_price_guard"),
	}
}

func (rg *ReferencePriceGuard) Enabled() bool {
	return rg != nil && rg.enabled
}

// ShouldSubmit compares reportedPrice with a reference price.
// Returns (shouldSubmit, reason, deviation, error).
// If the reference source cannot provide a value, this guard fails open.
func (rg *ReferencePriceGuard) ShouldSubmit(ctx context.Context, queryData []byte, reportedPrice float64) (bool, string, float64, error) {
	if !rg.Enabled() {
		return true, "", 0, nil
	}

	referencePrice, err := rg.source.GetReferencePrice(ctx, queryData)
	if err != nil {
		rg.logger.Warn("Could not fetch reference price, skipping guard check", "error", err)
		return true, "", 0, nil
	}

	if referencePrice == 0 {
		rg.logger.Warn("Reference price is zero, skipping guard check")
		return true, "", 0, nil
	}

	deviation := math.Abs(reportedPrice-referencePrice) / referencePrice
	if deviation > rg.maxDeviation {
		reason := fmt.Sprintf(
			"deviation %.5f%% exceeds max deviation %.5f%% (reported: %.6f, reference: %.6f)",
			deviation*100, rg.maxDeviation*100, reportedPrice, referencePrice,
		)
		return false, reason, deviation, nil
	}

	return true, "", deviation, nil
}

// BlocksizeSource is a ReferencePriceSource implementation backed by
// Blocksize's JSON-RPC market data API.
//
// It maps Tellor query IDs to Blocksize tickers and fetches the latest VWAP
// price for the corresponding ticker.
type BlocksizeSource struct {
}

var blocksizeAPIURL = "https://data.blocksize.capital/marketdata/v1/api"

// GetReferencePrice implements [ReferencePriceSource].
func (b *BlocksizeSource) GetReferencePrice(ctx context.Context, queryData []byte) (float64, error) {
	// Convert raw query data to the canonical Tellor query ID used as the map key.
	// `queryData` is ABI-encoded payload; `QueryIDFromData` computes keccak256(queryData).
	pair, err := supportedQueryIdsStr.GetPair(queryData)
	if err != nil {
		return 0, fmt.Errorf("failed to get pair for query data: %w", err)
	}

	apiKey := os.Getenv("BLOCKSIZE_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("missing BLOCKSIZE_API_KEY env var")
	}

	reqPayload := struct {
		ID     int    `json:"id"`
		Method string `json:"method"`
		Params struct {
			Ticker string `json:"ticker"`
		} `json:"params"`
	}{
		ID:     100,
		Method: "vwap_latest",
	}
	reqPayload.Params.Ticker = pair

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal blocksize request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, blocksizeAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return 0, fmt.Errorf("failed to create blocksize request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("blocksize request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("blocksize returned non-200 status: %d", resp.StatusCode)
	}

	var rpcResp struct {
		Result struct {
			VWAP struct {
				Ticker string  `json:"ticker"`
				Price  float64 `json:"price"`
			} `json:"vwap"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return 0, fmt.Errorf("failed to decode blocksize response: %w", err)
	}

	if rpcResp.Error != nil {
		return 0, fmt.Errorf("blocksize rpc error (%d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if rpcResp.Result.VWAP.Price <= 0 {
		return 0, fmt.Errorf("invalid blocksize price for %s: %f", pair, rpcResp.Result.VWAP.Price)
	}

	return rpcResp.Result.VWAP.Price, nil
}

var _ ReferencePriceSource = &BlocksizeSource{}

func NewBlocksizeSource() (*BlocksizeSource, error) {

	return &BlocksizeSource{}, nil
}

// QueryIds is a map of query data hex to pair name
type QueryIds map[string]string

func (q QueryIds) GetPair(queryData []byte) (string, error) {
	queryDataHex := hex.EncodeToString(utils.QueryIDFromData(queryData))
	pair, ok := q[queryDataHex]
	if !ok {
		return "", fmt.Errorf("pair not found for query id: %s", queryDataHex)
	}
	return pair, nil
}

var supportedQueryIdsStr = QueryIds{
	"83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992": "ETHUSD",
	"a6f013ee236804827b77696d350e9f0ac3e879328f2a3021d473a0b778ad78ac": "BTCUSD",
	"74c9cfdfd2e4a00a9437bf93bf6051e18e604a976f3fa37faafe0bb5a039431d": "SAGAUSD",
	"8ee44cd434ed5b0e007eee581fbe0855336f3f84484e8d9989a620a4a49aa0f7": "USDCUSD",
	"68a37787e65e85768d4aa6e385fb15760d46df0f67a18ec032d8fd5848aca264": "USDTUSD",
	"c444759b83c7bb0f6694306e1f719e65679d48ad754a31d3a366856becf1e71e": "fBTCUSD",
	"e010d752f28dcd2804004d0b57ab1bdc4eca092895d49160204120af11d15f3e": "USDNUSD",
	"59ae85cec665c779f18255dd4f3d97821e6a122691ee070b9a26888bc2a0e45a": "sUSDSUSD",
	"35155b44678db9e9e021c2cf49dd20c31b49e03415325c2beffb5221cf63882d": "yUSDUSD",
	"76b504e33305a63a3b80686c0b7bb99e7697466927ba78e224728e80bfaaa0be": "tBTCUSD",
	"0bc2d41117ae8779da7623ee76a109c88b84b9bf4d9b404524df04f7d0ca4ca7": "rETHUSD",
	"1962cde2f19178fe2bb2229e78a6d386e6406979edc7b9a1966d89d83b3ebf2e": "wstETHUSD",
	"d62f132d9d04dde6e223d4366c48b47cd9f90228acdc6fa755dab93266db5176": "KINGUSD",
	"03731257e35c49e44b267640126358e5decebdd8f18b5e8f229542ec86e318cf": "sUSDeUSD",
	"1808ab696a0b68ef13d854e21e188c794a3544a1746ac5d802d1a7d7a765cb51": "ATOMUSD",
	"5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0": "TRBUSD",
	"611fd0e88850bf0cc036d96d04d47605c90b993485c2971e022b5751bbb04f23": "stATOMUSD",
}
