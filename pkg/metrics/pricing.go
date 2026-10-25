package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultPricingEndpoint = "https://openrouter.ai/api/v1/models"

// ModelRate defines the dynamically resolved per-token pricing in USD.
type ModelRate struct {
	InputCostPerToken  float64 `json:"input_cost_per_token"`
	OutputCostPerToken float64 `json:"output_cost_per_token"`
}

// PricingRegistry manages live-fetched and user-configured model rates in memory.
// There are ZERO hardcoded model rates; all values are retrieved dynamically via API or user config.
type PricingRegistry struct {
	mu          sync.RWMutex
	liveRates   map[string]ModelRate
	customRates map[string]ModelRate
	httpClient  *http.Client
	lastSynced  time.Time
}

var globalRegistry = NewPricingRegistry()

// NewPricingRegistry creates an in-memory pricing registry.
func NewPricingRegistry() *PricingRegistry {
	return &PricingRegistry{
		liveRates:   make(map[string]ModelRate),
		customRates: make(map[string]ModelRate),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// FetchLiveRates queries the public pricing registry API and populates rates in RAM.
func (pr *PricingRegistry) FetchLiveRates(ctx context.Context) (int, error) {
	apiURL := os.Getenv("SUKAMI_PRICING_API_URL")
	if apiURL == "" {
		apiURL = os.Getenv("OPENROUTER_MODELS_URL")
	}
	if apiURL == "" {
		apiURL = defaultPricingEndpoint
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "Sukami-Gateway-Pricing/1.0")

	resp, err := pr.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("pricing API returned HTTP %d", resp.StatusCode)
	}

	var payload struct {
		Data []struct {
			ID      string `json:"id"`
			Pricing struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	count := 0
	for _, item := range payload.Data {
		promptRate, pErr := strconv.ParseFloat(item.Pricing.Prompt, 64)
		compRate, cErr := strconv.ParseFloat(item.Pricing.Completion, 64)
		if pErr != nil || cErr != nil || promptRate < 0 || compRate < 0 {
			continue
		}

		rate := ModelRate{
			InputCostPerToken:  promptRate,
			OutputCostPerToken: compRate,
		}

		fullKey := strings.ToLower(strings.TrimSpace(item.ID))
		pr.liveRates[fullKey] = rate

		// Also index clean name without prefix (e.g., "gpt-4o" for "openai/gpt-4o")
		if idx := strings.LastIndex(fullKey, "/"); idx != -1 {
			shortKey := fullKey[idx+1:]
			pr.liveRates[shortKey] = rate
		}
		count++
	}

	pr.lastSynced = time.Now()
	return count, nil
}

// SetCustomRate allows users or configurations to register custom negotiated rates.
func (pr *PricingRegistry) SetCustomRate(model string, inputCost, outputCost float64) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	cleaned := strings.ToLower(strings.TrimSpace(model))
	pr.customRates[cleaned] = ModelRate{
		InputCostPerToken:  inputCost,
		OutputCostPerToken: outputCost,
	}
}

// ResolveModelRate resolves the rate for a given model dynamically.
func (pr *PricingRegistry) ResolveModelRate(modelName string) (ModelRate, bool) {
	cleaned := strings.ToLower(strings.TrimSpace(modelName))
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	// 1. User/Enterprise custom rates take highest precedence
	if rate, exists := pr.customRates[cleaned]; exists {
		return rate, true
	}

	// 2. Direct match in live API fetched rates
	if rate, exists := pr.liveRates[cleaned]; exists {
		return rate, true
	}

	// 3. Short model name match
	if idx := strings.LastIndex(cleaned, "/"); idx != -1 {
		shortName := cleaned[idx+1:]
		if rate, exists := pr.liveRates[shortName]; exists {
			return rate, true
		}
	}

	// 4. Prefix/substring match for versioned models (e.g. "gpt-4o-2024-08-06")
	for key, rate := range pr.liveRates {
		if strings.HasPrefix(cleaned, key) || strings.Contains(cleaned, key) {
			return rate, true
		}
	}

	return ModelRate{}, false
}

// CalculateSavingsUSD computes exact dollar savings if a live or custom rate exists.
// Returns (0.0, false) if rate is uncataloged or offline, avoiding fabricated estimates.
func (pr *PricingRegistry) CalculateSavingsUSD(model string, inputSaved, outputSaved int) (float64, bool) {
	if inputSaved <= 0 && outputSaved <= 0 {
		return 0.0, true
	}
	rate, found := pr.ResolveModelRate(model)
	if !found {
		return 0.0, false
	}
	total := (float64(inputSaved) * rate.InputCostPerToken) + (float64(outputSaved) * rate.OutputCostPerToken)
	return total, true
}

// Global package-level wrappers forwarding to globalRegistry

// FetchLiveRates fetches dynamic pricing from upstream API.
func FetchLiveRates(ctx context.Context) (int, error) {
	return globalRegistry.FetchLiveRates(ctx)
}

// SetCustomRate sets a custom user/enterprise rate.
func SetCustomRate(model string, inputCost, outputCost float64) {
	globalRegistry.SetCustomRate(model, inputCost, outputCost)
}

// ResolveModelRate resolves rate for a model from dynamic registry.
func ResolveModelRate(modelName string) (ModelRate, bool) {
	return globalRegistry.ResolveModelRate(modelName)
}

// CalculateSavingsUSD computes exact dollar savings without hardcoded assumptions.
func CalculateSavingsUSD(model string, inputSaved, outputSaved int) (float64, bool) {
	return globalRegistry.CalculateSavingsUSD(model, inputSaved, outputSaved)
}
