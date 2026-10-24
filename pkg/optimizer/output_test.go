package optimizer

import (
	"strings"
	"sukami/pkg/models"
	"testing"
)

func TestApplyOutputOptimization_NoSystemMessage(t *testing.T) {
	msgs := []models.ChatMessage{
		{Role: "user", Content: "Write a high-performance Go worker pool"},
	}

	optimized, applied := ApplyOutputOptimization(msgs)
	if !applied {
		t.Fatal("expected applied to be true")
	}
	if len(optimized) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(optimized))
	}
	if optimized[0].Role != "system" {
		t.Fatalf("expected first message to be system, got %s", optimized[0].Role)
	}
	if !strings.Contains(optimized[0].Content, "Output Policy:") {
		t.Errorf("system message missing Output Policy directive")
	}
}

func TestApplyOutputOptimization_WithExistingSystemMessage(t *testing.T) {
	msgs := []models.ChatMessage{
		{Role: "system", Content: "You are an expert Go architect."},
		{Role: "user", Content: "How to handle graceful shutdown?"},
	}

	optimized, applied := ApplyOutputOptimization(msgs)
	if !applied {
		t.Fatal("expected applied to be true")
	}
	if len(optimized) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(optimized))
	}
	if !strings.Contains(optimized[0].Content, "You are an expert Go architect.") {
		t.Errorf("original system message content was lost")
	}
	if !strings.Contains(optimized[0].Content, "Output Policy:") {
		t.Errorf("expected appended Output Policy directive")
	}
}

func TestCleanOutputFluff(t *testing.T) {
	fluffy := "Certainly! Here is the solution you asked for:\n\npackage main\n\nfunc main() {}\n\nHope this helps! Let me know if you have any other questions!"
	cleaned, saved := CleanOutputFluff(fluffy)

	if strings.Contains(cleaned, "Certainly!") {
		t.Errorf("expected leading fluff to be removed, got: %s", cleaned)
	}
	if strings.Contains(cleaned, "Hope this helps!") {
		t.Errorf("expected trailing fluff to be removed, got: %s", cleaned)
	}
	if !strings.Contains(cleaned, "package main") {
		t.Errorf("expected core content to be preserved")
	}
	if saved <= 0 {
		t.Errorf("expected saved tokens > 0, got %d", saved)
	}
}

func TestCalculateOutputCostSaved(t *testing.T) {
	// 100,000 tokens saved at Claude Sonnet rate ($15/M)
	savings := CalculateOutputCostSaved(100000, 15.0)
	expected := 1.50
	if savings < expected-0.01 || savings > expected+0.01 {
		t.Errorf("expected $1.50, got $%.2f", savings)
	}
}
