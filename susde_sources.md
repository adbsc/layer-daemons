# sUSDE Price Sources

## Overview

sUSDE (staked USDe) is Ethena's staked version of USDe. It earns yield from Ethena's
staking rewards and auto-compounds. It can be swapped back to USDe on Ethena's DEX.

---

## SagaEVM (Chain ID: 5464)

**sUSDe on SagaEVM:**
| Token | Address | Holders | Transfers | Total Supply |
|-------|---------|---------|-----------|--------------|
| sUSDe | `0xE8EDdAAd31F250099691f93A186859bF39944c5d` | 1 | 1 | 0.378 |

**SagaEVM RPC:** `https://sagaevm.jsonrpc.sagarpc.io`
**SagaEVM Explorer:** `https://sagaevm.sagaexplorer.io` (Blockscout)

**Note:** sUSDe has very minimal adoption on SagaEVM (1 holder, 1 transfer, 0.378 total supply).
No active DEX pools found for sUSDe on Saga.

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
| Blockscout Explorer | `sagaevm.sagaexplorer.io` | Token prices shown |
| Blockscout API | `sagaevm.sagaexplorer.io/api/v2/` | Returns 404 for most endpoints |
| DEXScreener | `api.dexscreener.com` | Saga chain NOT yet indexed |
| GeckoTerminal | `api.geckoterminal.com` | Saga chain NOT yet indexed |

---

## Ethereum sUSDE Trading Venues

### Ethena DEX (Primary Source)

sUSDE is primarily traded on Ethena's own DEX.

| Pool | Address | Chain |
|------|---------|-------|
| USDe/sUSDE | `0x4b5E827F4C0a1042272a11857a355dA1F4Ceebae` | Ethereum (Curve factory-stable-ng-371) |
| sUSDE/WETH | Various | Ethereum (Uniswap V3) |

**Ethena DEX API:**
```
GET https://api.ethena.fi/price/sUSDE
```

### DEXScreener

```
GET https://api.dexscreener.com/latest/dex/tokens/0x9d39A5de30e57443BfF2A8307A4256c8797a3497
```
Returns all trading pairs with USD prices, liquidity, and 24h volume.

### DeFiLlama Coins API

```
GET https://coins.llama.fi/prices/current/ethereum:0x9d39A5de30e57443BfF2A8307A4256c8797a3497
```
Returns price in USD directly with confidence score and timestamp.

### On-chain Sources

- Call `getVirtualPrice()` on Curve pool contracts
- Call `balances()` on Curve pools to get reserves and calculate spot price
- Call `slot0()` on Uniswap v3 pools for current tick/price
- Ethena price feed contracts

## Contract Addresses

| Token | Address | Notes |
|-------|---------|-------|
| USDe (Ethereum) | `0x4c9EDD5852cd905f086C759E8383e09t116831` | Main USDe contract |
| sUSDE (Ethereum) | `0x9d39A5de30e57443BfF2A8307A4256c8797a3497` | Main sUSDE contract |
| sUSDE (SagaEVM) | `0xE8EDdAAd31F250099691f93A186859bF39944c5d` | 1 holder, 0.378 total supply |
| USDC (SagaEVM) | `0xfc960C233B8E98e0Cf282e29BDE8d3f105fc24d5` | Main stablecoin on SagaEVM |

## Recommendation

**For Ethereum sUSDE pricing, use this priority order:**
1. **Ethena API** (`api.ethena.fi/price/sUSDE`) - Official price source.
2. **DeFiLlama Coins API** (`coins.llama.fi`) - Simple, direct USD price with confidence.
3. **DEXScreener API** (`api.dexscreener.com/latest/dex/tokens/ADDRESS`) - All pairs with USD prices.
4. **On-chain Curve pools** - Call `getVirtualPrice()` or `balances()` directly.

**For SagaEVM sUSDE pricing:**
- Minimal on-chain liquidity. No reliable DEX pricing available yet.
- Consider using Ethereum sUSDE prices as reference until Saga liquidity matures.
