package client

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/joho/godotenv"
	"github.com/tellor-io/layer/utils"
)

func loadRepoRootDotEnv() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		_ = godotenv.Load(".env")
		return
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	_ = godotenv.Load(filepath.Join(repoRoot, ".env"))
}

func encodeSpotPriceQueryData(t *testing.T, asset, quote string) []byte {
	t.Helper()
	stringType, err := abi.NewType("string", "", nil)
	if err != nil {
		t.Fatalf("NewType string: %v", err)
	}
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		t.Fatalf("NewType bytes: %v", err)
	}
	innerArgs := abi.Arguments{{Type: stringType}, {Type: stringType}}
	inner, err := innerArgs.Pack(asset, quote)
	if err != nil {
		t.Fatalf("pack inner: %v", err)
	}
	outerArgs := abi.Arguments{{Type: stringType}, {Type: bytesType}}
	out, err := outerArgs.Pack("SpotPrice", inner)
	if err != nil {
		t.Fatalf("pack outer: %v", err)
	}
	return out
}

type mockReferencePriceSource struct {
	price float64
	err   error
}

func (m mockReferencePriceSource) GetReferencePrice(_ context.Context, _ []byte) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.price, nil
}

func TestReferencePriceGuard_DisabledWithoutProvider(t *testing.T) {
	logger := log.NewNopLogger()
	guard := NewReferencePriceGuard(nil, 0.1, logger)
	if guard.Enabled() {
		t.Fatalf("expected guard to be disabled without provider")
	}
}

func TestReferencePriceGuard_ShouldSubmit_WhenDisabled(t *testing.T) {
	logger := log.NewNopLogger()
	guard := NewReferencePriceGuard(nil, 0.1, logger)

	shouldSubmit, reason, deviation, err := guard.ShouldSubmit(context.Background(), []byte("q"), 123.45)
	if err != nil {
		t.Fatalf("expected no error when guard is disabled, got: %v", err)
	}
	if !shouldSubmit {
		t.Fatalf("expected submission to be allowed when guard is disabled")
	}
	if reason != "" {
		t.Fatalf("expected empty reason when guard is disabled, got: %q", reason)
	}
	if deviation != 0 {
		t.Fatalf("expected zero deviation when guard is disabled, got: %f", deviation)
	}
}

func TestReferencePriceGuard_DisabledWithoutDeviation(t *testing.T) {
	logger := log.NewNopLogger()
	guard := NewReferencePriceGuard(mockReferencePriceSource{price: 100}, 0, logger)
	if guard.Enabled() {
		t.Fatalf("expected guard to be disabled with zero max deviation")
	}
}

func TestReferencePriceGuard_WithinDeviationAllowed(t *testing.T) {
	logger := log.NewNopLogger()
	guard := NewReferencePriceGuard(mockReferencePriceSource{price: 100}, 0.1, logger)

	shouldSubmit, reason, deviation, err := guard.ShouldSubmit(context.Background(), []byte("q"), 105) // 5%
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shouldSubmit {
		t.Fatalf("expected submission to be allowed, reason: %s", reason)
	}
	if deviation <= 0 {
		t.Fatalf("expected positive deviation value, got: %f", deviation)
	}
}

func TestReferencePriceGuard_ExceedsDeviationBlocked(t *testing.T) {
	logger := log.NewNopLogger()
	guard := NewReferencePriceGuard(mockReferencePriceSource{price: 100}, 0.1, logger)

	shouldSubmit, reason, deviation, err := guard.ShouldSubmit(context.Background(), []byte("q"), 130) // 30%
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shouldSubmit {
		t.Fatalf("expected submission to be blocked")
	}
	if reason == "" {
		t.Fatalf("expected non-empty block reason")
	}
	if deviation <= 0 {
		t.Fatalf("expected positive deviation value, got: %f", deviation)
	}
}

func TestReferencePriceGuard_ReferenceSourceErrorFailsOpen(t *testing.T) {
	logger := log.NewNopLogger()
	guard := NewReferencePriceGuard(mockReferencePriceSource{err: errors.New("boom")}, 0.1, logger)

	shouldSubmit, reason, deviation, err := guard.ShouldSubmit(context.Background(), []byte("q"), 130)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shouldSubmit {
		t.Fatalf("expected fail-open allow, reason: %s", reason)
	}
	if deviation != 0 {
		t.Fatalf("expected zero deviation when source errors, got: %f", deviation)
	}
}

func TestReferencePriceGuard_ZeroReferenceFailsOpen(t *testing.T) {
	logger := log.NewNopLogger()
	guard := NewReferencePriceGuard(mockReferencePriceSource{price: 0}, 0.1, logger)

	shouldSubmit, reason, deviation, err := guard.ShouldSubmit(context.Background(), []byte("q"), 130)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shouldSubmit {
		t.Fatalf("expected zero-reference fail-open allow, reason: %s", reason)
	}
	if deviation != 0 {
		t.Fatalf("expected zero deviation for zero reference price, got: %f", deviation)
	}
}

// TestBlocksizeSource_GetReferencePrice calls the live Blocksize API. It loads
// BLOCKSIZE_API_KEY from the repository root .env (see loadRepoRootDotEnv) or
// from the environment, and skips when the key is absent so CI stays green.
func TestBlocksizeSource_GetReferencePrice(t *testing.T) {
	loadRepoRootDotEnv()
	if os.Getenv("BLOCKSIZE_API_KEY") == "" {
		t.Skip("BLOCKSIZE_API_KEY unset; add it to the repo .env or export it for this integration test")
	}

	src, err := NewBlocksizeSource()
	if err != nil {
		t.Fatalf("NewBlocksizeSource: %v", err)
	}

	qd := encodeSpotPriceQueryData(t, "eth", "usd")
	wantQueryID := "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"
	if got := hex.EncodeToString(utils.QueryIDFromData(qd)); got != wantQueryID {
		t.Fatalf("unexpected query id %s (want %s for ETH/USD SpotPrice)", got, wantQueryID)
	}

	price, err := src.GetReferencePrice(context.Background(), qd)
	if err != nil {
		t.Fatalf("GetReferencePrice: %v", err)
	}
	if price < 100 || price > 1_000_000 {
		t.Fatalf("ETH/USD reference price %f outside sanity bounds", price)
	}
}
