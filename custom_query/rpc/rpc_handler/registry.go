package rpchandler

import "fmt"

// curveFactoryPriceHandler is shared so legacy handler name curve_susde_factory_price stays compatible.
var curveFactoryPriceHandler = &CurveFactoryPriceHandler{}

var HandlerRegistry = map[string]RpcHandler{
	"generic":                        &GenericHandler{},
	"osmosis_pool_price_handler":     &OsmosisPoolPriceHandler{},
	"curve_factory_price":            curveFactoryPriceHandler,
	"curve_susde_factory_price":      curveFactoryPriceHandler,
	"subgraph_uniswap_pool_pair_usd": &SubgraphUniswapPoolPairHandler{},
}

func GetHandler(name string) (RpcHandler, error) {
	handler, exists := HandlerRegistry[name]
	if !exists {
		return nil, fmt.Errorf("unknown RPC handler: %s", name)
	}
	return handler, nil
}
