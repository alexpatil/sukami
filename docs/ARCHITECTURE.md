# Sukami Architecture Specification

Sukami is a high-performance, zero-dependency AI Token Optimization Gateway and Reverse Proxy written in pure, idiomatic Go standard library.

---

## 1. Request Lifecycle Flowchart

```text
[ Client (cURL / Python / Node / OpenAI SDK) ]
                      │
                      ▼
         [ CORS & Middleware Guard ]
                      │
                      ▼
   [ Gateway & Tenant Auth Guard ] ──────────(Unauthorized)────────► [ HTTP 401 Unauthorized ]
   - Cloud Security Mode: checks SUKAMI_API_KEY
   - Multi-tenant: validates X-Sukami-Key rate limits
                      │ (Authorized / Local Dev Mode)
                      ▼
   [ Upstream Provider Key Validation ] ────(No Provider Key)──────► [ HTTP 401 Missing API Key ]
   - Zero Cache Leak: verifies upstream key
     before evaluating cache or routes
                      │ (Valid Upstream Auth)
                      ▼
       [ Smart Prompt Complexity Router ]
          Evaluates prompt in <10µs.
          If low-complexity & opted-in:
          Routes gpt-4o -> gpt-4o-mini (~94% cost savings)
                      │
                      ▼
           [ Prompt Pruner Engine ]
          Strips filler, collapses whitespace.
          Unconditionally preserves code blocks & markdown.
                      │
                      ▼
        [ Semantic Vector Cache Check ] ───(Hit > 0.85 Sim)────────► [ HTTP 200 Instant (<10ms) ]
                      │ (Miss)
                      ▼
         [ Output Density Injection ]
          If X-Sukami-Density: true,
          injects anti-fluff directive into system prompt.
                      │
                      ▼
         [ Multi-Provider Dynamic Router ]
          Auto-routes by model/provider to:
          OpenAI · Groq · DeepSeek · OpenRouter · Ollama
                      │
                      ▼
      [ Upstream HTTP Forward / SSE Stream ]
                      │
                      ▼
       [ Metrics & Prometheus Exposition ]
          Records ground-truth tokens, latency,
          and live dynamic USD cost savings.
                      │
                      ▼
         [ Response Returned to Client ]
```

> **Transparent Proxy & Cloud Security Mode:**
> - **Local Developer Mode (Default):** Zero authentication friction. Point apps to `http://localhost:8080/v1` using standard upstream provider credentials (`Authorization: Bearer $UPSTREAM_API_KEY`).
> - **Cloud Security Mode (`SUKAMI_API_KEY`):** When deployed on remote cloud servers or public networks, set `SUKAMI_API_KEY=your-secret` to lock down the gateway. All requests require `X-Sukami-Master-Key` or valid tenant keys, and unauthenticated callers are rejected with `401 Unauthorized` before accessing cache or upstream providers (Zero Cache Leak guarantee). Multi-tenant keys (`X-Sukami-Key`) provide segregated metrics and rate limits.

---

## 2. Package Architecture

| Package | Responsibility | Concurrency & Storage Model |
| :--- | :--- | :--- |
| `pkg/auth` | Optional multi-tenant tenant management (`X-Sukami-Key`), rate limiting, and usage tracking | In-memory thread-safe `sync.RWMutex` store with per-minute rate limiter |
| `pkg/cache` | Multi-tier response caching: SHA-256 hash + Vector Cosine similarity with collision guards | In-memory bounded cache (10,000 max entries) with TTL & capacity eviction |
| `pkg/config` | Standard library environment loader (`.env`) | Pure Go key-value parser with comment and quote handling |
| `pkg/metrics` | Real-time Prometheus metrics exporter (`/metrics`), live dynamic pricing sync, request recorder | Lock-free / mutex-protected ring buffer (sliding window of recent requests) |
| `pkg/models` | OpenAI-compatible request, response, and SSE stream data types | Standard Go structs with JSON tags |
| `pkg/optimizer` | Input prompt pruning, output density directives, sub-10µs prompt complexity classifier, token estimation | Pure Go string scanners, compiled regexes, zero-allocation lexing |
| `pkg/proxy` | OpenAI-compatible reverse proxy endpoints (`/v1/chat/completions`, `/v1/models`, `/metrics`, `/health`, Root JSON descriptor) | Modern Go `http.ServeMux` with SSE streaming and upstream connection pooling |

---

## 3. Core Architectural Principles

1. **Zero External Dependencies**: `go.mod` contains 0 third-party packages. Everything is built with the Go standard library (`net/http`, `sync`, `regexp`, `crypto/rand`).
2. **Strict File Length Limits**: Every Go source file is kept strictly under 250 lines (`AGENTS.md` Rule 8) for readability and maintainability.
3. **Interface Segregation (SOLID)**: Handlers interact exclusively through interfaces (`Cache`, `KeyStore`, `RequestTracker`), making subsystems independently testable and swappable.
4. **Sub-10ms Overhead**: Heuristic lexical analysis runs in under 10 microseconds, prompt pruning executes in under 50 microseconds, and cache hits respond in sub-10ms.

---

## 4. Smart Model Routing & Complexity Classifier

Sukami features an in-process, zero-overhead lexical and syntactic prompt complexity classifier (`pkg/optimizer/classifier.go`). It evaluates incoming prompts in `<10µs` without calling any external model.

### Complexity Signals

| Signal Type | Triggers & Heuristics | Classification |
| :--- | :--- | :--- |
| **Code Syntax & Blocks** | Markdown fences (` ``` `), `func `, `struct `, `class `, `SELECT `, `goroutine`, stack traces (`panic:`, `NullPointerException`) | **Flagship Tier** |
| **Deep Reasoning** | "step by step", "mathematical proof", "architectural tradeoff", "distributed consensus", "memory leak" | **Flagship Tier** |
| **Long Context** | Prompt length exceeding 2,000 characters (~500+ tokens) | **Flagship Tier** |
| **Conversational / Factual** | Factual definitions, translation, grammar corrections, brief conversational questions | **Economy Tier** |

### Dynamic Model Tier Mapping

When Smart Routing is enabled (`X-Sukami-Smart-Route: true` or `ENABLE_SMART_ROUTING=true`), economy-classified prompts requesting expensive models are transparently routed to their lightweight counterparts:

| Original Model | Economy Routed Model | Cost Savings |
| :--- | :--- | :--- |
| `gpt-4o` | `gpt-4o-mini` | **~94%** |
| `gpt-4` / `gpt-4-turbo` | `gpt-4o-mini` | **~98%** |
| `claude-3-5-sonnet` | `claude-3-5-haiku` | **~80%** |
| `llama-3.3-70b-versatile` | `llama-3.1-8b-instant` | **~87%** |
| `deepseek-reasoner` | `deepseek-chat` | **~75%** |

### Transparent Telemetry Headers

Every smart-routed response includes transparent diagnostic headers so client applications retain full auditable visibility into model selection:

```http
X-Sukami-Original-Model: gpt-4o
X-Sukami-Routed-Model: gpt-4o-mini
X-Sukami-Route-Tier: economy
X-Sukami-Route-Reason: low-complexity prompt routed to gpt-4o-mini (~94% cost savings)
```
