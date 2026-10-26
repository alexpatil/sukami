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

// TestSemanticCosine_OppositeActionsPreventFalsePositives ensures opposite actions are NEVER matched.
func TestSemanticCosine_OppositeActionsPreventFalsePositives(t *testing.T) {
	testCases := []struct {
		name    string
		promptA string
		promptB string
	}{
		{"Backup vs Delete", "How to backup PostgreSQL database safely?", "How to delete PostgreSQL database safely?"},
		{"Kill vs Start", "How do I kill the background process?", "How do I start the background process?"},
		{"Disable vs Enable", "Disable auth in local configuration", "Enable auth in local configuration"},
		{"Encrypt vs Decrypt", "How to encrypt file using AES-256", "How to decrypt file using AES-256"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !HasConflictingIntent(tc.promptA, tc.promptB) {
				t.Errorf("Safety guard failed! %q vs %q must trigger conflict detection.", tc.promptA, tc.promptB)
			}
		})
	}
}

// TestSemanticCosine_EmptyAndWhitespace verifies that empty inputs yield 0 similarity without panic.
func TestSemanticCosine_EmptyAndWhitespace(t *testing.T) {
	vecEmpty := BuildSemanticVector("")
	vecSpace := BuildSemanticVector("   \n\t  ")
	vecReal := BuildSemanticVector("Explain Go channels")

	if sim := CosineSimilarity(vecEmpty, vecReal); sim != 0.0 {
		t.Errorf("Expected 0 similarity for empty vector, got %f", sim)
	}
	if sim := CosineSimilarity(vecSpace, vecReal); sim != 0.0 {
		t.Errorf("Expected 0 similarity for whitespace vector, got %f", sim)
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

	// 3. Technical entity collision (Docker vs Kubernetes) must MISS despite identical surrounding boilerplate
	dockerPrompt := "Please give me a complete guide on how to setup and configure Docker on Ubuntu 22.04 server"
	k8sPrompt := "Please give me a complete guide on how to setup and configure Kubernetes on Ubuntu 22.04 server"
	c.SetSemantic(ComputeHash("gpt-4o", dockerPrompt), dockerPrompt, []byte("Docker response"), "gpt-4o")
	_, hitCollision, _ := c.GetSemantic("gpt-4o", k8sPrompt, 0.80)
	if hitCollision {
		t.Fatalf("Entity collision occurred! Docker cached response was wrongly served for Kubernetes query.")
	}
}

// TestHasEntityConflict verifies domain entity conflict detection.
func TestHasEntityConflict(t *testing.T) {
	testCases := []struct {
		name     string
		textA    string
		textB    string
		conflict bool
	}{
		{
			name:     "Containers: Docker vs Kubernetes",
			textA:    "How to install Docker on Ubuntu",
			textB:    "How to install Kubernetes on Ubuntu",
			conflict: true,
		},
		{
			name:     "Databases: Postgres vs MySQL",
			textA:    "Connect to PostgreSQL in Go",
			textB:    "Connect to MySQL in Go",
			conflict: true,
		},
		{
			name:     "Languages: Python vs Ruby",
			textA:    "How to read a file in Python line by line",
			textB:    "How to read a file in Ruby line by line",
			conflict: true,
		},
		{
			name:     "Same technology with synonyms: Golang vs Go",
			textA:    "How to format strings in Go",
			textB:    "How to format strings in Golang",
			conflict: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := HasEntityConflict(tc.textA, tc.textB)
			if got != tc.conflict {
				t.Errorf("[%s] Expected conflict=%v, got=%v", tc.name, tc.conflict, got)
			}
		})
	}
}
