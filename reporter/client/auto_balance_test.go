package client

import (
	"testing"

	"github.com/stretchr/testify/require"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"
)

func TestNormalizeAutoBalanceEthAddr(t *testing.T) {
	addr, err := normalizeAutoBalanceEthAddr("0x0000000000000000000000000000000000000001")
	require.NoError(t, err)
	require.Equal(t, "0000000000000000000000000000000000000001", addr)

	addr, err = normalizeAutoBalanceEthAddr("0000000000000000000000000000000000000001")
	require.NoError(t, err)
	require.Equal(t, "0000000000000000000000000000000000000001", addr)

	addr, err = normalizeAutoBalanceEthAddr("")
	require.NoError(t, err)
	require.Empty(t, addr)

	_, err = normalizeAutoBalanceEthAddr("not-an-address")
	require.Error(t, err)
}

func TestParseAutoBalanceExecutionTime(t *testing.T) {
	hour, minute, err := parseAutoBalanceExecutionTime("03:05")
	require.NoError(t, err)
	require.Equal(t, 3, hour)
	require.Equal(t, 5, minute)

	_, _, err = parseAutoBalanceExecutionTime("24:00")
	require.Error(t, err)

	_, _, err = parseAutoBalanceExecutionTime("03")
	require.Error(t, err)
}

func TestIsBridgeDepositReportMsg(t *testing.T) {
	require.True(t, isBridgeDepositReportMsg(&oracletypes.MsgSubmitValue{}))
	require.False(t, isBridgeDepositReportMsg(&bridgetypes.MsgWithdrawTokens{}))
}

func TestShouldSkipAutoUnbond(t *testing.T) {
	reporterStake := math.LegacyNewDec(1_000)
	unbondAmount := math.NewInt(100)
	tenPercent, err := math.LegacyNewDecFromStr("0.10")
	require.NoError(t, err)
	fivePercent, err := math.LegacyNewDecFromStr("0.05")
	require.NoError(t, err)

	require.False(t, shouldSkipAutoUnbond(reporterStake, math.LegacyZeroDec(), unbondAmount), "zero max percentage disables the cap")
	require.False(t, shouldSkipAutoUnbond(reporterStake, tenPercent, unbondAmount))
	require.True(t, shouldSkipAutoUnbond(reporterStake, fivePercent, unbondAmount))
}
