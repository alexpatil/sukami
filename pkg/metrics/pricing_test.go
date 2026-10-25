package metrics

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestPricingRegistryCustom(t *testing.T) {
	registry := NewPricingRegistry()

	// 1. Unregistered model returns false (no fake hallucinated pricing)
	_, found := registry.ResolveModelRate("unregistered-xyz")
	if found {
		t.Fatalf("Expected unregistered model to return found=false")
	}

	// 2. Set custom rate
	registry.SetCustomRate("custom-model", 0.000002, 0.000008)

	rate, found := registry.ResolveModelRate("custom-model")
	if !found {
		t.Fatalf("Expected custom-model to be found")
	}
	if rate.InputCostPerToken != 0.000002 || rate.OutputCostPerToken != 0.000008 {
		t.Errorf("Unexpected rates: %+v", rate)
	}

	// 3. Exact savings calculation (1,000,000 input tokens = $2.00)
	usd, ok := registry.CalculateSavingsUSD("custom-model", 1000000, 0)
	if !ok || math.Abs(usd-2.00) > 1e-6 {
		t.Errorf("CalculateSavingsUSD = (%v, %v); want (2.00, true)", usd, ok)
	}
}

func TestFetchLiveRatesIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := FetchLiveRates(ctx)
	if err != nil {
		t.Skipf("Skipping live API fetch test (network/proxy restricted): %v", err)
	}
	if count < 50 {
		t.Errorf("FetchLiveRates loaded %d models; expected at least 50", count)
	}

	// Verify gpt-4o rate was dynamically resolved from API
	rate, found := ResolveModelRate("gpt-4o")
	if !found || rate.InputCostPerToken <= 0 {
		t.Errorf("gpt-4o rate not loaded dynamically: %+v, found: %v", rate, found)
	}
}
