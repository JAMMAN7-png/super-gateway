# Super AI Gateway

Self-hosted LLM aggregator combining **14+ providers** (9 free-tier, 5 paid) behind a single OpenAI-compatible `/v1/chat/completions` endpoint. Multi-key rotation, SOCKS5/HTTP proxy support, automatic fallback chains, response caching, and built-in SearXNG web search.

## Architecture

```
Client (OpenAI SDK)
    │
    ▼
┌─────────────────────────────────────────┐
│         Super AI Gateway :3000           │
│  /v1/chat/completions  /v1/models       │
│  /v1/search  /v1/stats  /health         │
├─────────────────────────────────────────┤
│  FALLBACK ENGINE (free → paid last)     │
│  ┌──────┐ ┌───────────┐ ┌────────────┐  │
│  │ Key  │ │  Circuit  │ │  Response  │  │
│  │ Pool │ │  Breaker  │ │  Cache     │  │
│  └──────┘ └───────────┘ └────────────┘  │
├─────────────────────────────────────────┤
│  PROVIDER ADAPTERS                      │
│  Groq  Cerebras  DeepSeek  Cloudflare   │
│  Novita  Chutes  NVIDIA  Ollama         │
│  OpenRouter  OpenCode  NanoGPT  Vercel  │
├─────────────────────────────────────────┤
│  SearXNG :8080  │  SOCKS5/HTTP Proxy    │
└─────────────────────────────────────────┘
```

## Quick Start

### 1. Get API Keys

Sign up for the free tiers (no credit card required for most):

| Provider | Signup | Free Limits | 
|----------|--------|-------------|
| Groq | https://console.groq.com | 30 RPM, 6K TPM per key |
| Cerebras | https://cloud.cerebras.ai | 30 RPM, 1M tokens/day |
| DeepSeek | https://platform.deepseek.com | 1M tokens/day |
| Cloudflare | https://dash.cloudflare.com/ai/workers-ai | 10K neurons/day |
| Novita | https://novita.ai | $0.50 credit + 60 RPM |
| Chutes | https://chutes.ai | 500K tokens/day |
| NVIDIA | https://build.nvidia.com | Free (Dev Program) |
| OpenRouter | https://openrouter.ai | Free models available |
| OpenCode | https://opencode.ai | Free models available |

### 2. Configure

```bash
cp .env.example .env
# Edit .env — add your API keys
# For multi-key: GROQ_API_KEY=key1,key2,key3

cp config.example.yaml config.yaml
# Edit config.yaml if needed
```

### 3. Deploy

```bash
# Start SearXNG for web search
mkdir -p searxng
cp searxng/settings.yml searxng/
cp searxng/limiter.toml searxng/

# Build and run
docker compose up -d
```

### 4. Use It

```bash
# Test with curl
curl http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "free-fast",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'

# Use with any OpenAI SDK — just change base_url
```

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:3000/v1",
    api_key="not-needed"  # gateway handles keys internally
)

response = client.chat.completions.create(
    model="free-fast",
    messages=[{"role": "user", "content": "What is Rust?"}]
)
print(response.choices[0].message.content)
```

## Meta-Models

Built-in virtual models that auto-route to the best provider:

| Meta-Model | Strategy | Falls back through |
|------------|----------|--------------------|
| `free-fast` | Lowest latency | Groq → Cerebras → DeepSeek |
| `free-smart` | Most capable | Cerebras 120B → Groq 70B → Chutes 72B |
| `fallback-all` | Exhaustive | All 10 free providers → paid |

Add custom meta-models in `config.yaml`:

```yaml
meta_models:
  my-custom:
    strategy: ordered
    models:
      - "groq/llama-3.3-70b-versatile"
      - "cerebras/gpt-oss-120b"
      - "openrouter/meta-llama/llama-4-scout:free"
```

## Key Rotation & Proxy

**Multi-key rotation**: Set `GROQ_API_KEY=key1,key2,key3` and the gateway spreads requests across keys. When one hits rate limit (429), it's cooled down and the next key is used automatically.

**SOCKS5/HTTP proxy**: Set `global_proxy: "socks5://user:pass@proxy:1080"` in config.yaml, or per-provider with the `proxy` field. This is essential for:

1. **Bypassing regional blocks** — DeepSeek, some Chinese providers geo-restrict
2. **Creating multiple accounts** — Each key from a different IP avoids rate limit correlation
3. **DataImpulse** is recommended: $1/GB residential, traffic never expires, no subscription

## Web Search

```bash
# Direct search
curl -X POST http://localhost:3000/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "latest AI research 2026", "limit": 5}'

# Integrated with chat — use the search endpoint programmatically:
# 1. Search via /v1/search
# 2. Inject results into your prompt
# 3. Send to /v1/chat/completions
```

## Fallback Behavior

The gateway tries providers in this order:

1. **Same-model free providers** (ordered by priority in config)
2. **Cross-model free providers** (if model not found, tries similar models)
3. **Paid providers** (only if `paid_only_after_free_failure: true`)

Each provider-level attempt cycles through available keys with exponential cooldown on 429 errors. After all keys for a provider are exhausted, the next provider is tried.

Circuit breakers prevent hammering dead providers — after 5 consecutive failures, the provider is blocked for 60 seconds.

## Rate Limit Math

With 3 keys per provider and the current free tiers:

| Scaled Up | Keys | Effective Free RPM | Daily Requests |
|-----------|------|-------------------|----------------|
| Groq × 3 | 3 | ~90 RPM | ~3,000/day |
| Cerebras × 3 | 3 | ~90 RPM | ~3,000/day |
| DeepSeek × 1 | 1 | ~60 RPM | ~1M tokens/day |
| Cloudflare × 1 | 1 | ~40 RPM | 10K req/day |
| Novita × 1 | 1 | 60 RPM | credit-based |
| Chutes × 1 | 1 | varies | 500K tokens/day |
| **Total** | **10** | **~340 RPM** | **generous** |

With proxy rotation → effectively unlimited free-tier accounts → unlimited free requests.

## API Reference

### POST /v1/chat/completions
OpenAI-compatible chat completion. Supports streaming (`"stream": true`).

### GET /v1/models
Lists all available models across all providers, including meta-models.

### POST /v1/search
Web search via SearXNG. Body: `{"query": "...", "engines": ["google"], "limit": 8}`

### GET /v1/stats
Gateway statistics: provider counts, available keys, cache hit ratio.

### GET /health
Health check.

## Performance

- **Go + Fiber**: ~50K RPS on modest hardware (vs LiteLLM Python at ~500 RPS before P99 degrades)
- **Response caching**: SHA-256 prompt hashing, ~23ms cache hits vs 500-2000ms cold path
- **LRU eviction**: Configurable max entries with automatic cleanup
- **Connection pooling**: 100 idle connections, 20 per host

## Comparison to Alternatives

| Feature | Super Gateway | FreeLLM | LiteLLM | OpenRouter |
|---------|--------------|---------|---------|------------|
| Free-tier optimized | ✅ | ✅ | ❌ | ❌ |
| Multi-key rotation | ✅ | ✅ | Enterprise only | ❌ |
| SOCKS5 proxy | ✅ | ❌ | ❌ | ❌ |
| Web search built-in | ✅ | ❌ | Plugin | ❌ |
| Circuit breakers | ✅ | ✅ | Partial | ✅ |
| Response caching | ✅ | ✅ | Plugin | ❌ |
| Self-hosted | ✅ | ✅ | ✅ | ❌ |
| Language | Go | TS | Python | SaaS |
| Meta-models | ✅ | ✅ | ❌ | ❌ |
| Paid provider fallback | ✅ | ❌ | ✅ | ✅ |

## License

MIT
