package rpchandler

import (
	"context"
	"time"

	reader "github.com/tellor-io/layer-daemons/custom_query/rpc/rpc_reader"
	pricefeedservertypes "github.com/tellor-io/layer-daemons/server/types/pricefeed"
)

type RpcHandler interface {
	FetchValue(
		ctx context.Context,
		client *reader.Reader,
		invert bool,
		usdViaID uint32,
		priceCache *pricefeedservertypes.MarketToExchangePrices,
		maxDataAge time.Duration,
	) (float64, error)
}
