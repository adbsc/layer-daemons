# sUSDE Price Sources

## Current Implementation
- On-chain RPC: `convertToAssets()` on sUSDE contract (0x9D39A5DE30e57443BfF2A8307A4256c8797A3497)
- Multiplied by cached USDE/USD price from pricefeed

## sUSDE Trading Venues

### DeFiLlama Coins API (RECOMMENDED - Simplest)

**Tested working, returns direct USD price with confidence score:**
```
GET https://coins.llama.fi/prices/current/ethereum:0x9D39A5DE30e57443BfF2A8307A4256c8797A3497
```
Response:
```json
{
  "coins": {
    "ethereum:0x9D39A5DE30e57443BfF2A8307A4256c8797A3497": {
      "decimals": 18,
      "price": 1.2300754311482356,
      "symbol": "sUSDe",
      "timestamp": 1778523535,
      "confidence": 1
    }
  }
}
```
Note: The older `api.llama.fi/prices/current/...` endpoint was returning HTTP errors/empty responses.
Use `coins.llama.fi` instead.

### DEXScreener API (RECOMMENDED - Comprehensive)

**Tested working, returns all pairs with USD prices, liquidity, and volume:**
```
GET https://api.dexscreener.com/latest/dex/tokens/0x9D39A5DE30e57443BfF2A8307A4256c8797A3497
```
Returns all trading pairs with `priceUsd`, `liquidity.usd`, and `volume.h24` fields.
No authentication needed. Good for cross-validation and monitoring multiple venues.

### Curve Finance Pools (Primary On-chain Source)

All sUSDE pools are on Curve's factory-stable-ng factory.

| Pool | ID | Address | Coins | TVL | sUSDe USD Price |
|------|----|---------|-------|-----|-----------------|
| sUSDe/DOLA | factory-stable-ng-xxx | `0x744793B5110f6ca9cC7CDfe1CE16677c3Eb192ef` | sUSDe, DOLA | $61M | $1.23 |
| sDAI/sUSDe | factory-stable-ng-102 | `0x167478921b907422F8E88B43C4Af2B8BEa278d3A` | sDAI, sUSDe | $7M | $1.17 |
| sUSD/sUSDe | factory-stable-ng-371 | `0x4b5E827F4C0a1042272a11857a355dA1F4Ceebae` | sUSD (old), sUSDe | $2.9M | $0.57 (sUSD price) |
| scrvUSD/sUSDe | factory-stable-ng-xxx | `0xd29f8980852c2c76fC3f6E96a7Aa06E0BedCC1B1` | scrvUSD, sUSDe | $2.7M | $1.10 |
| sUSDe/reUSD | factory-stable-ng-xxx | `0x5C2ab69Eb2BF12A2f4572D178687Bd4660512972` | sUSDe, reUSD | $1.4M | $1.23 |
| sUSDe/crvUSD | factory-stable-ng-xxx | `0x57064F49Ad7123C92560882a45518374ad982e85` | sUSDe, crvUSD | $782K | $1.22 |
| reUSDe/sUSDe | factory-twocrypto-xxx | `0x43b98EEA5C689F0036918f590a4B55f22D853734` | reUSDe, sUSDe | $548K | $1.34 |
| sUSDe/USDT | factory-stable-ng-613 | `0xA9335f627bA9D80d9DDa76b5FeA03dD885404132` | USDT, sUSDe | (varies) | $1.23 |
| sUSDe/sUSDS | factory-stable-ng-370 | `0x3CEf1AFC0E8324b57293a6E7cE663781bbEFBB79` | sUSDe, sUSDS | (varies) | $1.23 |
| FRAX/sUSDe | factory-stable-ng-xxx | (from Ethena docs) | FRAX, sUSDe | (varies) | $1.23 |
| sUSDe/crvUSD | factory-stable-ng-xxx | (from Curve API) | sUSDe, crvUSD | (varies) | $1.23 |
| USD3/sUSDe | factory-stable-ng-xxx | (from Curve API) | USD3, sUSDe | (varies) | $1.23 |
| DOLA/sUSDe | factory-stable-ng-xxx | (from Curve API) | DOLA, sUSDe | (varies) | $1.23 |
| MUSD/sUSDe | factory-stable-ng-xxx | (from Curve API) | MUSD, sUSDe | (varies) | $1.23 |
| reUSD/sUSDe | factory-stable-ng-xxx | (from Curve API) | reUSD, sUSDe | (varies) | $1.23 |

**Note:** The sUSD/sUSDe pool reports the sUSD price (~$0.57), NOT the sUSDe price.
Use pools with USD-pegged stablecoins (USDT, USDC, DAI, etc.) for accurate sUSDe pricing.

#### Working API

**Curve Pools List (RECOMMENDED - tested working):**
```
GET https://api.curve.fi/api/getPools/ethereum/factory-stable-ng
```
Returns all factory-stable-ng pools with real-time USD prices. Filter for pools containing sUSDe.
Response includes `coins[].usdPrice` for each token in each pool.

**Also check factory-twocrypto:**
```
GET https://api.curve.fi/api/getPools/ethereum/factory-twocrypto
```

**Individual Pool Endpoints (behind Cloudflare, may require browser JS):**
```
GET https://api.curve.fi/api/getPoolInfo/ethereum/factory-stable-ng/{id}
GET https://api.curve.fi/api/getPoolPrice/ethereum/factory-stable-ng/{id}/{coinIndex}
```

### Uniswap v3

**sUSDe/USDT Pools (highest volume):**
- Pool: `0xb20351bcf606dcc3525d2ed36760a86a5dec7423b77d41125bd4a416ba93448b` - TVL: $12.2M, 24h vol: $8.95M
- Pool: `0xeac10ed910ed8b167bd5785954d53641705dceb46c4fd077f7c265bef455ffb8` - TVL: $6.78M
- Pool: `0x7EB59373D63627be64b42406B108B602174B4CCC` - TVL: $432K

**USDT/USDe Pool:**
- Pool: `0x435664008f38b0650fbc1c9fc971d0a3bc2f1e47`
- Subgraph: `https://api.thegraph.com/subgraphs/name/uniswap/uniswap-v3`
- Query: `{ pool(id: "0x435664008f38b0650fbc1c9fc971d0a3bc2f1e47") { token0Price token1Price } }`
- Note: subgraph may return empty from certain endpoints, needs verification

### DeFiLlama Legacy API (flaky/404 from some endpoints)
```
GET https://api.llama.fi/prices/current/ethereum:0x9D39A5DE30e57443BfF2A8307A4256c8797A3497
```
Returns price in USD directly. Was returning HTTP error/empty from our endpoint - may need different network path.
**Use `coins.llama.fi` instead** (see above).

### Ethena API (requires whitelisting)
```
GET https://public.api.ethena.fi/asset-availability
GET https://public.api.ethena.fi/rfq?pair=USDT%2FUSDe&side=MINT&size=100000&benefactor=0x...
```
RFQ response includes exchange rate between USDT and USDe. Not directly sUSDE but useful.

### On-chain Alternatives
- Call `getVirtualPrice()` on Curve pool contracts directly
- Call `balances()` on Curve pools to get reserves and calculate spot price
- Call `getReserves()` on Uniswap v2 pool contracts
- Call `slot0()` on Uniswap v3 pools for current tick/price

## Contract Addresses
- sUSDE: `0x9D39A5DE30e57443BfF2A8307A4256c8797A3497`
- USDe: `0x4c9EDD5852cd905f086C759E8383e09bff1E68B3`

## Recommendation

Use this priority order for sUSDE/USD pricing:

1. **DeFiLlama Coins API** (`coins.llama.fi`) - Simplest, returns direct USD price with confidence=1.0.
2. **Curve API** (`api.curve.fi/api/getPools/ethereum/factory-stable-ng`) - Returns real-time prices from the most liquid pools. Use the `coins[].usdPrice` field.
3. **DEXScreener API** (`api.dexscreener.com/latest/dex/tokens/ADDRESS`) - Returns all pairs with USD prices, liquidity, and volume. Good for cross-validation.
4. **On-chain Curve pools** - Call `getVirtualPrice()` or `balances()` directly on pool contracts for trustless pricing.
