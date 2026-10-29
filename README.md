# Sukami

A lightweight, zero-dependency caching reverse proxy for LLM APIs, written in pure Go.

It sits between your application and your LLM provider. When you make an API request, Sukami trims conversational filler, checks an in-memory vector cache for repeated queries, and forwards the rest directly to your upstream provider.

---

## Highlights

- **Fast semantic cache**: Serves repeated queries from in-memory cache in under 10ms with guardrails against false matches.
- **Prompt pruning**: Strips common conversational filler ("Please kindly...", redundant whitespace) without touching code blocks or markdown indentation.
- **Drop-in replacement**: Compatible with OpenAI SDKs, LangChain, or cURL by changing only the base URL.
- **Zero dependencies**: Single static Go binary (~10 MB) built exclusively with the Go standard library.

---

## Quickstart

### 1. Start the proxy

```bash
# Clone the repository
git clone https://github.com/alexpatil/sukami.git
cd sukami

# Run with your upstream provider URL
go run ./cmd/sukami -upstream https://api.openai.com/v1
```

Or with Docker:

```bash
docker compose up -d
```

The gateway starts at `http://localhost:8080`.

---

## Usage

Point your existing OpenAI client to `http://localhost:8080/v1`:

### Python

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-upstream-api-key",
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "How do channels work in Go?"}],
)
print(response.choices[0].message.content)
```

### TypeScript / Node.js

```typescript
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "http://localhost:8080/v1",
  apiKey: "your-upstream-api-key",
});

const response = await client.chat.completions.create({
  model: "gpt-4o",
  messages: [{ role: "user", content: "How do channels work in Go?" }],
});
console.log(response.choices[0].message.content);
```

### cURL

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $UPSTREAM_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "How do channels work in Go?"}]
  }'
```

Responses are standard OpenAI JSON objects with an added `X-Cache` header (`HIT` or `MISS`).

---

## How It Works

```text
Client Application ──► Sukami (:8080) ──► Upstream Provider
                             │
                      ┌──────┴──────────────┐
                      │ 1. Verify auth      │
                      │ 2. Prune filler     │
                      │ 3. Check cache      │
                      │ 4. Forward upstream │
                      └─────────────────────┘
```

1. **Authentication**: Verifies credentials and passes your API key directly to the upstream provider.
2. **Prompt pruning**: Strips common filler phrases. Code blocks, indentation, and structure remain untouched.
3. **Cache check**: If a matching response is in memory, Sukami responds instantly (`X-Cache: HIT`).
4. **Upstream forward**: On a cache miss, the request is forwarded to your configured provider, and the response is cached for next time.

---

## Configuration

Set options via environment variables or command-line flags:

| Variable | Flag | Default | Description |
| :--- | :--- | :--- | :--- |
| `PORT` | `-port`, `-p` | `8080` | Port to listen on |
| `UPSTREAM_URL` | `-upstream` | *(none)* | Upstream provider URL (e.g. `https://api.openai.com/v1`) |
| `UPSTREAM_API_KEY` | | *(none)* | Default upstream API key if not supplied by the client |
| `SUKAMI_API_KEY` | | *(none)* | Master gateway key to lock down the proxy in production |
| `ENABLE_SMART_ROUTING` | | `false` | Routes simple queries to economy models (e.g. gpt-4o-mini) |
| `ENABLE_OUTPUT_DENSITY` | | `false` | Adds anti-fluff instructions to system prompts |

### Per-Request Headers

You can also override settings per request:

- `X-Upstream-Url: https://...` — Routes this request to a custom upstream endpoint.
- `X-Sukami-Smart-Route: true` — Enables smart model routing for this request.
- `X-Sukami-Density: true` — Requests concise output without conversational padding.

---

## Endpoints

| Method | Path | Description |
| :--- | :--- | :--- |
| `POST` | `/v1/chat/completions` | OpenAI-compatible completions endpoint (supports streaming SSE) |
| `GET` | `/v1/models` | Transparently returns available models from upstream |
| `GET` | `/metrics` | Standard Prometheus metrics (requests, cache hits, tokens saved) |
| `GET` | `/health` | Simple health check endpoint |
| `GET` | `/` | Service status JSON |

---

## Development

```bash
# Run tests
go test ./pkg/...

# Run static analysis
go vet ./...

# Build binary
go build -o bin/sukami ./cmd/sukami
```

---

## License

[MIT](LICENSE) © Ganesh Shinde
