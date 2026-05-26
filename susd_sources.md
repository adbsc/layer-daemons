# sUSD Price Sources

## Important: Two sUSD Contracts Exist (Ethereum)

MakerDAO's sUSD has been migrated. There are TWO active sUSD contracts with very different prices:

| Version | Address | Price | Status |
|---------|---------|-------|--------|
| OLD sUSD | `0x57Ab1ec28D129707052df4dF418D58a2D46d5f51` | ~$0.55-0.57 | Deprecated but actively traded on DEXes |
| NEW sUSD | `0x4F8E1426A9d10bddc11d26042ad270F16cCb95F2` | ~$0.99-1.00 | Current pegged version, minimal DEX liquidity |

**For DEX-based pricing, use the OLD sUSD address** -- it has $4M+ in total DEX liquidity.
The NEW sUSD has essentially no DEX pairs (only one empty Curve pool).

---

## SagaEVM (Chain ID: 5464)

**sUSD on SagaEVM:** No dedicated sUSD contract found on SagaEVM. sUSD is NOT deployed
natively on Saga. If trading sUSD on Saga, you're likely using a wrapped/bridged version.

**sUSDS on SagaEVM:**
| Token | Address | Holders | Transfers | Max Supply |
|-------|---------|---------|-----------|------------|
| sUSDS | `0x9f2013831e371587a8E39f2A43DF774af2178e35` | 10 | 14 | 1,000,000,000,000,000,000 |
| sUSDS (alt) | `0xd3D94a06fDcCDEdf7C0e32BD4EF71dFfeff3f982` | 2 | - | - |

**sUSDe on SagaEVM:**
| Token | Address | Holders | Transfers | Total Supply |
|-------|---------|---------|-----------|--------------|
| sUSDe | `0xE8EDdAAd31F250099691f93A186859bF39944c5d` | 1 | 1 | 0.378 |

**SagaEVM RPC:** `https://sagaevm.jsonrpc.sagarpc.io`
**SagaEVM Explorer:** `https://sagaevm.sagaexplorer.io` (Blockscout)

**Note:** Both sUSDS and sUSDe have very minimal adoption on SagaEVM (single-digit holders,
minimal transfers). No active DEX pools found for these tokens on Saga.

### SagaEVM DEXes

- **Uniswap V3** - deployed on SagaEVM. SwapRouter02 at `0x8cf7FfAA54718BDbA047A8613C2a7799655D5491`
- **Colt** - Saga-native DEX (registered on DeFiLlama). Peak TVL ~$6M, dropped to $0 as of Jan 2026.
- **Palomino Finance** - Saga-native (DeFiLlama registered)
- **Mustang Finance** - Saga-native (DeFiLlama registered)
- **Beefy** - yield aggregator on Saga
- **Steer Protocol** - liquidity management on Saga
- **YieldFi** - yield optimization on Saga

### SagaEVM Price Data Sources

| Source | Endpoint | Status |
|--------|----------|--------|
| DeFiLlama Chains | `api.llama.fi/chains` | Saga registered (chainId: 5464, TVL: $0) |
| DeFiLlama Protocol | `api.llama.fi/protocol/colt` | TVL data available |
| Blockscout Explorer | `sagaevm.sagaexplorer.io` | Token prices shown (USDC at $1.000) |
| Blockscout API | `sagaevm.sagaexplorer.io/api/v2/` | Returns 404 for most endpoints |
| DEXScreener | `api.dexscreener.com` | Saga chain NOT yet indexed |
| GeckoTerminal | `api.geckoterminal.com` | Saga chain NOT yet indexed |

---

## Ethereum sUSD Trading Venues

### Curve Finance Pools (Primary Source)

All sUSD pools are on Curve's factory-stable-ng factory.

| Pool | ID | Address | Coins | TVL | sUSD USD Price |
|------|----|---------|-------|-----|----------------|
| sUSD/sUSDe | factory-stable-ng-371 | `0x4b5E827F4C0a1042272a11857a355dA1F4Ceebae` | sUSD (old), sUSDe | $2.8M | $0.555 |
| sUSD/USDe | factory-stable-ng-346 | `0x59a06b97b2d566B9Dee2a368EaC8787Cfa57f95D` | sUSD (old), USDe | $89 | $0.555 |
| 7Synth | factory-stable-ng-656 | `0x458e273312D2692ACf12C7A1bE67D1e9ebCbe7C0` | 8 synthetic currencies | $0 | $0.555 |

#### Working API

**Curve Pools List (RECOMMENDED - tested working):**
```
GET https://api.curve.fi/api/getPools/ethereum/factory-stable-ng
```
Returns all factory-stable-ng pools with real-time USD prices. Filter for pools containing sUSD
at address `0x57Ab1ec28D129707052df4dF418D58a2D46d5f51`.
Response includes `coins[].usdPrice` for each token in each pool.

### DEXScreener (Tested Working - Returns Direct USD Price)

**Direct Token Price:**
```
GET https://api.dexscreener.com/latest/dex/tokens/0x57Ab1ec28D129707052df4dF418D58a2D46d5f51
```
Returns all trading pairs with USD prices, liquidity, and 24h volume. The `priceUsd` field
on each pair gives the direct sUSD/USD price. No authentication needed.

### DeFiLlama Coins API (Tested Working)

```
GET https://coins.llama.fi/prices/current/ethereum:0x57Ab1ec28D129707052df4dF418D58a2D46d5f51
```
Returns price in USD directly with confidence score and timestamp.
Response format:
```json
{
  "coins": {
    "ethereum:0x57Ab1ec28D129707052df4dF418D58a2D46d5f51": {
      "decimals": 18,
      "symbol": "sUSD",
      "price": 0.5719105607383518,
      "timestamp": 1778524286,
      "confidence": 0.99
    }
  }
}
```

### Uniswap v3 / Sushiswap Pairs (via DEXScreener)

| Pair | DEX | TVL | Pair Address |
|------|-----|-----|--------------|
| sUSD/SNX | Uniswap | $1.07M | `0xA3ccaf08a54Cf31649f91aE1570A0720C8d4EB1E` |
| sUSD/WETH | Uniswap | $125K | `0xf80758aB42C3B07dA84053Fd88804bCB6BAA4b5c` |
| sUSD/WETH | SushiSwap | $113K | `0xF1F85b2C54a2bD284B1cf4141D64fD171Bd85539` |
| sUSD/USDC | Uniswap | $99K | `0x9899D32bc818be6F4b4772bF8B1eeD70fEb10BD2` |

### On-chain Alternatives
- Call `getVirtualPrice()` on Curve pool contracts directly
- Call `balances()` on Curve pools to get reserves and calculate spot price
- Call `getReserves()` on Uniswap v2 pair contracts to calculate spot price
- Call `slot0()` on Uniswap v3 pools for current tick/price

---

## Contract Addresses

| Token | Address | Notes |
|-------|---------|-------|
| OLD sUSD (Ethereum) | `0x57Ab1ec28D129707052df4dF418D58a2D46d5f51` | Deprecated, ~$0.55 price, $4M+ DEX liquidity |
| NEW sUSD (Ethereum) | `0x4F8E1426A9d10bddc11d26042ad270F16cCb95F2` | Current, ~$1.00 price, minimal DEX liquidity |
| sUSDS (SagaEVM) | `0x9f2013831e371587a8E39f2A43DF774af2178e35` | 10 holders, minimal activity |
| sUSDe (SagaEVM) | `0xE8EDdAAd31F250099691f93A186859bF39944c5d` | 1 holder, 0.378 total supply |

## Recommendation

**For Ethereum sUSD pricing, use this priority order:**
1. **DeFiLlama Coins API** (`coins.llama.fi`) - Simplest, returns direct USD price with confidence.
2. **Curve API** (`api.curve.fi/api/getPools/ethereum/factory-stable-ng`) - Real-time prices from most liquid pool.
3. **DEXScreener API** (`api.dexscreener.com/latest/dex/tokens/ADDRESS`) - All pairs with USD prices, liquidity, volume.
4. **On-chain Curve pools** - Call `getVirtualPrice()` or `balances()` directly on pool contracts.

**For SagaEVM sUSD/sUSDS pricing:**
- Minimal on-chain liquidity. No reliable DEX pricing available yet.
- Blockscout explorer shows token data but no price feeds.
- Consider using Ethereum prices as reference until Saga liquidity matures.
