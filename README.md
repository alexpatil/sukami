<div align="center">

# Sukami

**High-Performance AI Gateway & Token Optimization Proxy in Pure Go**

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-6B7280?style=flat-square)](LICENSE)
[![Pure Go](https://img.shields.io/badge/dependencies-zero-4ADE80?style=flat-square)](go.mod)
[![Docker](https://img.shields.io/badge/Docker-~10MB-0EA5E9?style=flat-square&logo=docker&logoColor=white)](https://hub.docker.com)

</div>

---

Sukami is a drop-in OpenAI-compatible gateway that transparently reduces token consumption by 20–35% and serves semantically equivalent requests from a sub-10ms in-memory vector cache—without modifying a single line of application code.

It is a single static binary. Zero runtime dependencies. No YAML configuration required.

---

## Quickstart

**Go**

```bash
go run ./cmd/sukami
```

**Docker**

```bash
docker compose up -d
```

The gateway starts on `http://localhost:8080`. No further configuration is required.

---

## Drop-in Integration

Change one line. Nothing else.

**Python (OpenAI SDK)**

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-...",
    base_url="http://localhost:8080/v1",  # only this line changes
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Compare PostgreSQL and MongoDB."}],
)
```

**cURL**

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Compare PostgreSQL and MongoDB."}]
  }'
```

The response is a standard OpenAI JSON object. No schema changes, no wrapper fields.

---

## How It Works

Sukami operates as a transparent middleware layer on every request.

**1. Transparent Prompt Pruning**
Strips conversational filler prefixes and redundant whitespace before the request reaches the upstream provider. Code blocks, markdown indentation, and prompt semantics are preserved unconditionally. Net result: 20–35% fewer input tokens billed.

**2. Sub-10ms Semantic Vector Cache**
Incoming prompts are embedded and matched against cached results using cosine similarity. Two conflict guards protect cache correctness:
- **Entity Collision Guard** — prevents false positives between distinct technical subjects (e.g., Docker vs. Kubernetes).
- **Opposites Conflict Guard** — prevents cache hits across antonymic intent (e.g., `delete` vs. `backup`, `encrypt` vs. `decrypt`).

**3. Smart Multi-Provider Routing**
Routes automatically to OpenAI, Groq Cloud, DeepSeek, OpenRouter, or a local Ollama instance based on the requested model name. No routing rules to define.

**4. Native Prometheus Observability**
A built-in Prometheus 0.0.4-compatible exporter is available at `GET /metrics`. Tracks request volume, cache hit ratio, token savings, and real-time dollar savings computed from live model pricing.

---

## Benchmark

Workload: enterprise comparative technical evaluation (PostgreSQL vs. MongoDB).

| Path               | Output Tokens | Latency  | Cost vs. Direct |
|--------------------|---------------|----------|-----------------|
| Direct upstream    | 3,072         | 7.33s    | 100%            |
| Via Sukami         | 1,979         | 4.50s    | ~64% (−35.6%)   |
| Semantic cache hit | —             | < 10ms   | 0%              |

**38.6% faster. 35.6% fewer tokens. ~$750 saved per 100,000 queries.**

Reproduce with:

```bash
go run ./cmd/sukami-bench
```

---

## Observability

```
GET /metrics
```

Sukami exports standard Prometheus text format. Key metrics:

| Metric | Description |
|---|---|
| `sukami_requests_total` | Total proxied requests processed |
| `sukami_cache_hits_total` | Semantic cache hits served without upstream calls |
| `sukami_cache_hit_ratio` | Real-time semantic cache hit ratio (0.0 to 1.0) |
| `sukami_tokens_saved_total` | Cumulative input tokens eliminated by pruning |
| `sukami_cost_saved_usd_total` | Dollar savings computed dynamically from live model pricing |

Plug into any Prometheus-compatible stack—Grafana, Datadog Agent, VictoriaMetrics—with no additional configuration.

---

## Architecture

```
Client Request (OpenAI-compatible)
        │
        ▼
┌─────────────────────────────┐
│       Prompt Pruner         │  strips filler, preserves code & markdown
└────────────┬────────────────┘
             │
             ▼
┌─────────────────────────────┐
│     Semantic Vector Cache   │  cosine similarity + collision guards
└────────────┬────────────────┘
             │ miss
             ▼
┌─────────────────────────────┐
│    Multi-Provider Router    │  OpenAI · Groq · DeepSeek · OpenRouter · Ollama
└────────────┬────────────────┘
             │
             ▼
┌─────────────────────────────┐
│   Prometheus Metrics Layer  │  tokens · latency · cost
└─────────────────────────────┘
```

---

## Roadmap

Sukami is production-ready for single-node deployments. Planned capabilities:

- [ ] **Distributed Cache Sync** — Multi-node cluster synchronization via Redis / Valkey.
- [ ] **Team Budgets & Cost Caps** — Centralized department quotas and hard per-team spend limits.
- [ ] **PII & Data Redaction** — Automated sensitive data masking for HIPAA / SOC2 compliance.
- [ ] **Managed Cloud Gateway** — Dedicated zero-maintenance hosted offering.

Discuss features and share use cases in [GitHub Discussions](https://github.com/alexpatil/sukami/discussions).  
Direct inquiries: [codingpenin@gmail.com](mailto:codingpenin@gmail.com)

---

## License

MIT © [Ganesh Shinde](mailto:codingpenin@gmail.com)

See [LICENSE](LICENSE) for full terms.
