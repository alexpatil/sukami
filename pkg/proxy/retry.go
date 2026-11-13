package proxy

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultInitialBackoff = 250 * time.Millisecond
	defaultMaxBackoff     = 10 * time.Second
	maxAllowedRetries     = 5
)

// SetMaxRetries configures the default maximum retry attempts on transient upstream errors.
func (h *Handler) SetMaxRetries(n int) {
	if n >= 0 {
		h.maxRetries = n
	}
}

// resolveEffectiveMaxRetries checks per-request header and server-level defaults.
func (h *Handler) resolveEffectiveMaxRetries(r *http.Request) int {
	headerVal := r.Header.Get("X-Sukami-Max-Retries")
	if headerVal == "" {
		headerVal = r.Header.Get("X-Max-Retries")
	}
	if headerVal != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(headerVal)); err == nil && n >= 0 {
			if n > maxAllowedRetries {
				return maxAllowedRetries
			}
			return n
		}
	}
	if h.maxRetries > maxAllowedRetries {
		return maxAllowedRetries
	}
	return h.maxRetries
}

// isTransientStatusCode checks if an HTTP response code is retryable.
func isTransientStatusCode(code int) bool {
	return code == http.StatusTooManyRequests ||
		code == http.StatusBadGateway ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout
}

// isTransientNetworkError checks if a network error is retryable.
func isTransientNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection refused") ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF)
}

// parseRetryAfter extracts sleep duration from Retry-After or retry-after-ms header.
func parseRetryAfter(h http.Header) (time.Duration, bool) {
	if msStr := h.Get("retry-after-ms"); msStr != "" {
		if ms, err := strconv.Atoi(strings.TrimSpace(msStr)); err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond, true
		}
	}
	val := strings.TrimSpace(h.Get("Retry-After"))
	if val == "" {
		return 0, false
	}
	// Integer seconds
	if sec, err := strconv.Atoi(val); err == nil && sec >= 0 {
		return time.Duration(sec) * time.Second, true
	}
	// HTTP date
	if t, err := http.ParseTime(val); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d, true
		}
		return 0, true
	}
	return 0, false
}

// calculateBackoff computes exponential backoff with jitter and retry-after header support.
func calculateBackoff(attempt int, respHeader http.Header) time.Duration {
	if respHeader != nil {
		if dur, ok := parseRetryAfter(respHeader); ok && dur > 0 {
			if dur > defaultMaxBackoff {
				return defaultMaxBackoff
			}
			return dur
		}
	}

	backoffFloat := float64(defaultInitialBackoff) * math.Pow(2, float64(attempt-1))
	if backoffFloat > float64(defaultMaxBackoff) {
		backoffFloat = float64(defaultMaxBackoff)
	}

	// Add 0-25% random jitter to avoid thundering herd
	var b [8]byte
	_, _ = rand.Read(b[:])
	jitterRatio := float64(binary.LittleEndian.Uint64(b[:])%250) / 1000.0 // 0.000 to 0.250
	jittered := time.Duration(backoffFloat * (1.0 + jitterRatio))
	if jittered > defaultMaxBackoff {
		return defaultMaxBackoff
	}
	return jittered
}

// sleepWithContext pauses execution until duration elapses or context is cancelled.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// executeWithRetry performs an HTTP request with automatic retries on transient errors.
func (h *Handler) executeWithRetry(
	ctx context.Context,
	reqFactory func() (*http.Request, error),
	maxRetries int,
) (*http.Response, int, error) {
	var lastErr error
	retries := 0

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			return nil, retries, ctx.Err()
		}

		req, err := reqFactory()
		if err != nil {
			return nil, retries, err
		}

		resp, err := h.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if !isTransientNetworkError(err) || attempt >= maxRetries {
				return nil, retries, err
			}
			retries++
			backoff := calculateBackoff(retries, nil)
			slog.Warn("Transient upstream network error, retrying",
				"attempt", attempt+1,
				"backoff", backoff,
				"err", err,
			)
			if sleepErr := sleepWithContext(ctx, backoff); sleepErr != nil {
				return nil, retries, sleepErr
			}
			continue
		}

		if isTransientStatusCode(resp.StatusCode) && attempt < maxRetries {
			retries++
			backoff := calculateBackoff(retries, resp.Header)
			slog.Warn("Transient upstream status code, retrying",
				"status", resp.StatusCode,
				"attempt", attempt+1,
				"backoff", backoff,
			)
			_ = resp.Body.Close()
			if sleepErr := sleepWithContext(ctx, backoff); sleepErr != nil {
				return nil, retries, sleepErr
			}
			continue
		}

		return resp, retries, nil
	}

	return nil, retries, lastErr
}
