package client

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/tellor-io/layer-daemons/lib/metrics"
	"github.com/tellor-io/layer/utils"
)

// Metric labels for price guard.
const (
	LabelPair      = "pair"
	LabelDeviation = "deviation"
	LabelReason    = "reason"
)

// Block reasons for price guard.
const (
	ReasonDeviation = "deviation"
	ReasonExpired   = "expired"
)

var (
	priceGuardPairRegistry = make(map[string]string) // queryDataHex -> pair
	priceGuardRegistryMu   sync.RWMutex
)

// RegisterPriceGuardMarketParams registers market params for pair label lookup in metrics.
// Accepts any slice of structs with QueryData and Pair string fields.
func RegisterPriceGuardMarketParams(params any) {
	priceGuardRegistryMu.Lock()
	defer priceGuardRegistryMu.Unlock()

	priceGuardPairRegistry = make(map[string]string)
	v := reflect.ValueOf(params)
	if v.Kind() != reflect.Slice {
		return
	}
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		qd := item.FieldByName("QueryData")
		pair := item.FieldByName("Pair")
		if qd.IsValid() && pair.IsValid() {
			priceGuardPairRegistry[strings.ToLower(qd.String())] = pair.String()
		}
	}
}

func getPriceGuardPair(queryDataHex string) string {
	priceGuardRegistryMu.RLock()
	defer priceGuardRegistryMu.RUnlock()
	return priceGuardPairRegistry[strings.ToLower(queryDataHex)]
}

// emitPriceGuardSkippedMetric emits a counter metric each time a report is
// skipped by the price guard. reason should be one of ReasonDeviation or ReasonExpired.
func emitPriceGuardSkippedMetric(queryData []byte, deviationPercent float64, reason string) {
	queryIDBytes := utils.QueryIDFromData(queryData)
	queryDataStr := hex.EncodeToString(queryData)

	pair := getPriceGuardPair(queryDataStr)
	pairLabel := pair
	if pairLabel == "" {
		pairLabel = fmt.Sprintf("%x", queryIDBytes)
	}

	metrics.IncrCounterWithLabels(
		"daemon_report_skipped",
		1,
		metrics.Label{Name: LabelPair, Value: pairLabel},
		metrics.Label{Name: LabelDeviation, Value: fmt.Sprintf("%.2f%%", deviationPercent)},
		metrics.Label{Name: LabelReason, Value: reason},
	)
}
