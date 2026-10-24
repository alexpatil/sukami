# Sukami Architecture Specification

Sukami is an ultra-fast, zero-dependency AI Token Optimization Gateway and Reverse Proxy written in idiomatic Go 1.22+.

---

## 1. Request Lifecycle Flowchart

```text
[ Client (cURL / Python / Node) ]
               │
               ▼
   [ CORS & Middleware Guard ]
               │
               ▼
    [ Auth & Free Quota Guard ] ────(Exhausted)───► [ HTTP 402 Payment Required ]
               │
               ▼
     [ Prompt Pruning Engine ]  ───► Eliminates redundant conversational filler
               │
               ▼
    [ Semantic Vector Cache ]   ───(Hit > 0.85)───► [ HTTP 200 Instant (10ms) ]
               │ (Miss)
               ▼
  [ Output Density Injection ]  ───► Injects anti-fluff directive into system prompt
               │
               ▼
 [ Upstream LLM (Groq / OpenAI) ]
               │
               ▼
    [ Output Fluff Stripper ]   ───► Cleans residual conversational boilerplate
               │
               ▼
    [ Metrics & Quota Update ]  ───► Atomically tracks token savings & usage
               │
               ▼
[ Response Streamed to Client ]
```

---

## 2. Package Architecture

| Package | Responsibility | Concurrency & Storage Model |
| :--- | :--- | :--- |
| `pkg/auth` | API Key validation, Free trial 100k quota enforcement, Token-bucket rate limiting | In-Memory thread-safe `sync.RWMutex` map |
| `pkg/cache` | Multi-tier response caching: Exact hash + Vector Cosine similarity | LRU In-memory with TTL eviction and normalized term vectors |
| `pkg/optimizer` | Input prompt pruning (preserving code blocks) + Output density directives & fluff stripper | Zero-allocation string scanners and regex filters |
| `pkg/metrics` | Real-time observability, In/Out token savings breakdown, USD cost tracking | Ring-buffer sliding window (default 50 recent requests) |
| `pkg/proxy` | OpenAI-compatible `/v1/chat/completions` and `/v1/models` HTTP handlers | Go 1.22+ `http.ServeMux` with streaming SSE support |
| `web` | Single-Page Application (SPA) dashboard | React 19 + TypeScript + Vite, served directly from Go binary |

---

## 3. Design Principles (SOLID, KISS, DRY)

1. **Strict File Limits**: Every Go source file is kept under 250 lines for maximum maintainability.
2. **Interface Segregation**: Handlers interact with abstract `Cache`, `KeyStore`, and `RequestTracker` interfaces, decoupling storage implementations.
3. **Sub-10ms Overhead**: Zero heavy frameworks; stdlib `net/http` first.
