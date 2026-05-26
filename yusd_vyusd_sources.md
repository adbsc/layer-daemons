# yUSD and vyUSD Price Sources

## Overview

**yUSD** (YUSD Stablecoin) - Crypto-backed stablecoin by YieldNexus (yeti.finance). Primarily on Avalanche.

**vyUSD** (YieldFi vyUSD) - Yield-bearing vault token by YieldFi. Wraps yUSD and earns yield from the YieldFi protocol. Deployed on Ethereum L2s.

---

## Contract Addresses

| Token | Chain | Address | Notes |
|-------|-------|---------|-------|
| yUSD | Avalanche | `0x111111111111ed1D73f860F57b2798b683f2d325` | Primary deployment, 11.8M supply |
| yUSD | BSC | `0xAB3dBcD9B096C3fF76275038bf58eAC10D22C61f` | 1.8M supply, 1 DEX pair |
| yUSD | Ethereum | `0x4274cD7277C7bb0806Bd5FE84b9aDAE466a8DA0a` | Has Uniswap pairs but contract code=0x (possibly proxy or stale) |
| vyUSD | Ethereum | `0x2e3c5e514eef46727de1fe44618027a9b70d92fc` | CoinGecko listed, code=0x (possibly proxy) |
| vyUSD | Optimism | `0xf4f447e6afa04c9d11ef0e2fc0d7f19c24ee55de` | Code=1588 bytes, ERC4626 vault, 3,856 supply |
| vyUSD | Arbitrum | `0xf4f447e6afa04c9d11ef0e2fc0d7f19c24ee55de` | Same as Optimism |
| vyUSD | Base | `0xf4f447e6afa04c9d11ef0e2fc0d7f19c24ee55de` | Same as Optimism |
| vyUSD | Sonic | `0xf4f447e6afa04c9d11ef0e2fc0d7f19c24ee55de` | Same as Optimism |

### vyUSD on Optimism - Implementation

vyUSD on Optimism is a proxy with EIP-1967 implementation at:
`0x4ed3166fab585d9da1126955c9b1f0f61c971801`

It's an ERC4626 vault with `totalSupply() = 3,855.87`. Note that `convertToAssets(1e18)` returned 0 and `asset()` returned 0x0, which may indicate the vault is in a paused state or uses a custom implementation.

---

## yUSD Trading Venues

### DEXScreener (Avalanche - Primary)

**yUSD/USDC on KyberSwap:**
- Pair: `0xCF908d925b21594f9a92b264167A85B0649051a8`
- Price: ~$0.9967, Liquidity: ~$21

**yUSD/sAVAX on KyberSwap:**
- Price: ~$1.0022, Liquidity: ~$1,170

**yUSD/WAVAX on TraderJoe:**
- Price: ~$0.9953, Liquidity: ~$142

```
GET https://api.dexscreener.com/latest/dex/tokens/0x111111111111ed1D73f860F57b2798b683f2d325
```
Returns all pairs with `priceUsd`, `liquidity.usd`, and `volume.h24`.

### DEXScreener (BSC)

**yUSD/WBNB on PancakeSwap:**
- Price: ~$0.9883, Liquidity: ~$19,243

```
GET https://api.dexscreener.com/latest/dex/tokens/0xAB3dBcD9B096C3fF76275038bf58eAC10D22C61f
```

### DEXScreener (Ethereum)

Multiple yUSD pairs on Uniswap v3 (but token contract has no code - verify):

| Pair | DEX | Price | Liquidity |
|------|-----|-------|-----------|
| YUSD/USDC | Uniswap v3 | ~$0.9988 | ~$832,625 |
| YUSD/USDT | Uniswap v3 | ~$0.9988 | ~$589,301 |
| YUSD/USDT | Uniswap v3 | ~$0.9988 | ~$140,076 |
| YUSD/USDT | Curve | ~$0.9990 | ~$40,099 |
| YUSD/WETH | Uniswap v3 | ~$1.0012 | ~$1,182 |

```
GET https://api.dexscreener.com/latest/dex/tokens/0x4274cD7277C7bb0806Bd5FE84b9aDAE466a8DA0a
```

**IMPORTANT:** The Ethereum yUSD token contract has no deployed code (code=0x). The DEXScreener pairs may be from an older deployment or the token may use a minimal proxy pattern. Verify before using.

---

## vyUSD Trading Venues

### DEXScreener

vyUSD has **0 DEX pairs** across all chains on DEXScreener. The price is derived from the vault's internal pricing mechanism (DeFiLlama tracks it at $0.9236 with confidence=0.99).

---

## Price Data APIs

### DeFiLlama Coins API (RECOMMENDED)

**Tested working for both tokens:**

**yUSD (Avalanche):**
```
GET https://coins.llama.fi/prices/current/avalanche:0x111111111111ed1D73f860F57b2798b683f2d325
```
Response:
```json
{
  "coins": {
    "avalanche:0x111111111111ed1D73f860F57b2798b683f2d325": {
      "decimals": 18,
      "price": 0.9968800660395419,
      "symbol": "YUSD",
      "timestamp": 1778543944,
      "confidence": 0.99
    }
  }
}
```

**yUSD (BSC):**
```
GET https://coins.llama.fi/prices/current/bsc:0xAB3dBcD9B096C3fF76275038bf58eAC10D22C61f
```
Response: price ~$0.9986, confidence=0.99

**vyUSD (any L2):**
```
GET https://coins.llama.fi/prices/current/optimism:0xf4f447e6afa04c9d11ef0e2fc0d7f19c24ee55de
GET https://coins.llama.fi/prices/current/arbitrum:0xf4f447e6afa04c9d11ef0e2fc0d7f19c24ee55de
GET https://coins.llama.fi/prices/current/base:0xf4f447e6afa04c9d11ef0e2fc0d7f19c24ee55de
GET https://coins.llama.fi/prices/current/sonic:0xf4f447e6afa04c9d11ef0e2fc0d7f19c24ee55de
```
Response: price ~$0.923628, confidence=0.99

### CoinGecko API

**yUSD:**
```
GET https://api.coingecko.com/api/v3/simple/price?ids=yusd-stablecoin&vs_currencies=usd
```
Response: `{"yusd-stablecoin": {"usd": 0.996878}}`

**vyUSD:**
```
GET https://api.coingecko.com/api/v3/simple/price?ids=yieldfi-vyusd&vs_currencies=usd
```
Response: `{"yieldfi-vyusd": {"usd": 0.923628}}`

### Curve API

**yUSD on Avalanche:** No Curve pools found for yUSD on Avalanche.

```
GET https://api.curve.fi/api/getPools/avalanche/factory-stable-ng
GET https://api.curve.fi/api/getPools/avalanche/factory-crypto-ng
```

### DeFiLlama Protocol API

**YieldFi protocol data (includes yUSD and vyUSD TVL):**
```
GET https://api.llama.fi/protocol/yieldfi
```
Returns chain TVLs and token supply data over time. Includes YUSD and VYUSD token supplies.

**yeti-finance protocol data:**
```
GET https://api.llama.fi/protocol/yeti-finance
```

---

## On-chain Price Methods

### yUSD (Avalanche)

The yUSD contract on Avalanche (`0x111111111111ed1D73f860F57b2798b683f2d325`) is a standard ERC20 token (not an ERC4626 vault). `convertToAssets()` returns 0, meaning it's NOT a yield-bearing token.

- `name()`: "YUSD Stablecoin"
- `symbol()`: "YUSD"
- `decimals()`: 18
- `totalSupply()`: 11,837,376.57

For yUSD/USD pricing, use DEX pair prices or the APIs above.

### vyUSD (Optimism)

The vyUSD contract on Optimism (`0xf4f447e6afa04c9d11ef0e2fc0d7f19c24ee55de`) is an ERC4626 vault:

- `name()`: "YieldFi vyUSD"
- `symbol()`: "vyUSD"
- `decimals()`: 18
- `totalSupply()`: 3,855.87
- `asset()`: 0x000...000 (possibly paused or custom implementation)
- `convertToAssets(1e18)`: 0 (possibly paused)
- `managementFee()`: 0
- `performanceFee()`: 0

**NOTE:** The vault appears to be in a paused or transitional state. `convertToAssets()` and `asset()` both return 0. DeFiLlama still tracks the price at $0.923628, likely from historical data or an off-chain oracle.

---

## GeckoTerminal

**yUSD on Avalanche:** GeckoTerminal returns empty for yUSD on Avalanche (no indexed pairs).

```
GET https://api.geckoterminal.com/api/v2/simple/networks/avalanche/tokens/0x111111111111ed1D73f860F57b2798b683f2d325
```

---

## Protocol Details

### YieldNexus (yUSD)

- Website: https://yeti.finance/
- Docs: https://docs.yeti.finance/
- CoinGecko: yusd-stablecoin
- DeFiLlama protocol: yeti-finance (TVL: $0)
- yUSD is a crypto-backed stablecoin, primarily on Avalanche

### YieldFi (vyUSD)

- Website: https://yieldfi.xyz/
- CoinGecko: yieldfi-vyusd
- DeFiLlama protocol: yieldfi (TVL: ~$12M on Ethereum, smaller on L2s)
- vyUSD is a yield-bearing vault token that wraps yUSD
- YieldFi also has vBTC, vETH, and other vault tokens

---

## Recommendation

**For yUSD/USD pricing, use this priority order:**

1. **DeFiLlama Coins API** (`coins.llama.fi`) - Returns direct USD price with confidence. Works for both Avalanche and BSC deployments.
2. **CoinGecko API** (`api.coingecko.com/api/v3/simple/price?ids=yusd-stablecoin`) - Simple, direct USD price.
3. **DEXScreener API** (`api.dexscreener.com/latest/dex/tokens/ADDRESS`) - Returns all pairs with USD prices and liquidity. Best for Avalanche pairs.
4. **On-chain DEX prices** - Calculate from KyberSwap/TraderJoe pairs on Avalanche.

**For vyUSD/USD pricing, use this priority order:**

1. **DeFiLlama Coins API** (`coins.llama.fi`) - Primary source. Returns price ~$0.92 with confidence=0.99 for all L2 chains.
2. **CoinGecko API** (`api.coingecko.com/api/v3/simple/price?ids=yieldfi-vyusd`) - Direct USD price.
3. **On-chain vault methods** - Call `convertToAssets()` on the ERC4626 vault. NOTE: currently returns 0 on Optimism, may be paused.
4. **YieldFi protocol API** (`api.yieldfi.xyz`) - May have vault pricing (SSL issues from some endpoints).

**Key observations:**
- yUSD is primarily an Avalanche token with very limited DEX liquidity (< $15K total)
- vyUSD is primarily on Ethereum L2s with NO DEX pairs (price from DeFiLlama/CoinGecko only)
- Both tokens are relatively illiquid and may have stale prices
- vyUSD price (~$0.92) is significantly below the $1 peg, suggesting yield accrual or depeg
- Ethereum yUSD has DEX pairs on DEXScreener but the contract has no deployed code - verify before using
