package cache

import (
	"testing"
	"time"
)

// TestSemanticCosine_DifferentWordsSameMeaning tests semantic matching when words differ completely.
func TestSemanticCosine_DifferentWordsSameMeaning(t *testing.T) {
	prompt1 := "How do I reverse a string in Python?"
	prompt2 := "Write Python code to flip text backwards."

	vec1 := BuildSemanticVector(prompt1)
	vec2 := BuildSemanticVector(prompt2)

	sim := CosineSimilarity(vec1, vec2)
	t.Logf("Cosine similarity between %q and %q = %.4f", prompt1, prompt2, sim)

	// Threshold for semantic equivalence is >= 0.80
	if sim < 0.80 {
		t.Fatalf("Expected semantic similarity >= 0.80 for equivalent intent, got %.4f", sim)
	}

	if HasConflictingIntent(prompt1, prompt2) {
		t.Errorf("Should not have conflicting intent!")
	}
}

// TestSemanticCosine_OppositeActionsPreventFalsePositives ensures backup vs delete are NEVER matched.
func TestSemanticCosine_OppositeActionsPreventFalsePositives(t *testing.T) {
	promptBackup := "How to backup PostgreSQL database safely?"
	promptDelete := "How to delete PostgreSQL database safely?"

	if !HasConflictingIntent(promptBackup, promptDelete) {
		t.Fatalf("Safety guard failed! 'backup' vs 'delete' must trigger conflict detection.")
	}
}

// TestSemanticCosine_UnrelatedQueries ensures unrelated topics have near-zero similarity.
func TestSemanticCosine_UnrelatedQueries(t *testing.T) {
	promptA := "Explain quantum computing algorithms"
	promptB := "How to bake a chocolate cake recipe"

	vecA := BuildSemanticVector(promptA)
	vecB := BuildSemanticVector(promptB)

	sim := CosineSimilarity(vecA, vecB)
	t.Logf("Unrelated similarity = %.4f", sim)

	if sim > 0.15 {
		t.Fatalf("Unrelated prompts should have near-zero similarity, got %.4f", sim)
	}
}

// TestMemoryCache_GetSemantic_EndToEnd tests the complete semantic lookup cache.
func TestMemoryCache_GetSemantic_EndToEnd(t *testing.T) {
	c := NewMemoryCache(10 * time.Minute)

	originalQuery := "How do I reverse a string in Python?"
	storedResponse := []byte(`{"id":"chatcmpl-test","choices":[{"message":{"content":"Use s[::-1]"}}]}`)

	key := ComputeHash("gpt-4o", originalQuery)
	c.SetSemantic(key, originalQuery, storedResponse, "gpt-4o")

	// 1. Query with completely different phrasing
	differentPhrasing := "Write Python code to flip text backwards"
	retrieved, hit, similarity := c.GetSemantic("gpt-4o", differentPhrasing, 0.80)

	if !hit {
		t.Fatalf("Expected True Semantic Cache HIT for different words with same meaning!")
	}

	t.Logf("True Semantic Cache HIT! Similarity: %.2f%%", similarity*100)
	if string(retrieved) != string(storedResponse) {
		t.Fatalf("Response mismatch on semantic cache hit")
	}

	// 2. Dangerous opposite query must MISS despite similar phrasing
	c.SetSemantic(ComputeHash("gpt-4o", "backup database"), "backup database", []byte("backup response"), "gpt-4o")
	_, hitOpposite, _ := c.GetSemantic("gpt-4o", "delete database", 0.80)
	if hitOpposite {
		t.Fatalf("Dangerous opposite action caused False Positive! It should have been rejected.")
	}
}
