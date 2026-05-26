package contract_handlers

import (
	"context"
	"time"

	reader "github.com/tellor-io/layer-daemons/custom_query/contracts/contract_reader"
	pricefeedservertypes "github.com/tellor-io/layer-daemons/server/types/pricefeed"
)

type ContractHandler interface {
	FetchValue(ctx context.Context, client *reader.Reader, priceCache *pricefeedservertypes.MarketToExchangePrices, maxDataAge time.Duration) (float64, error)
}
