package cache

import (
	"math"
	"strings"
	"unicode"
)

// Vector represents a sparse term-frequency map for semantic comparisons.
type Vector map[string]float64

// stopWords references the unified boilerplate set from entities.go for consistent filtering.
var stopWords = boilerplateWords

// BuildSemanticVector constructs a clean term-frequency vector from prompt text.
func BuildSemanticVector(text string) Vector {
	vec := make(Vector)
	words := extractCleanWords(text)
	if len(words) == 0 {
		return vec
	}

	for _, w := range words {
		if stopWords[w] {
			continue // Filter out boilerplate noise
		}
		canonical := canonicalStem(w)
		vec[canonical] += 1.0
	}

	normalize(vec)
	return vec
}

// CosineSimilarity calculates the mathematical cosine similarity between two vectors.
func CosineSimilarity(a, b Vector) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	dotProduct := 0.0
	small, large := a, b
	if len(a) > len(b) {
		small, large = b, a
	}

	for term, valA := range small {
		if valB, found := large[term]; found {
			dotProduct += valA * valB
		}
	}

	if dotProduct > 1.0 {
		return 1.0
	}
	return dotProduct
}

// HasConflictingIntent prevents false positives on dangerous opposite actions (e.g. backup vs delete).
func HasConflictingIntent(textA, textB string) bool {
	a := strings.ToLower(textA)
	b := strings.ToLower(textB)

	criticalOpposites := [][2]string{
		{"delete", "backup"},
		{"remove", "backup"},
		{"drop", "restore"},
		{"kill", "start"},
		{"stop", "start"},
		{"disable", "enable"},
	}

	for _, pair := range criticalOpposites {
		hasA1, hasB2 := strings.Contains(a, pair[0]), strings.Contains(b, pair[1])
		hasA2, hasB1 := strings.Contains(a, pair[1]), strings.Contains(b, pair[0])
		if (hasA1 && hasB2) || (hasA2 && hasB1) {
			return true // Conflict detected!
		}
	}
	return false
}

func normalize(v Vector) {
	sumSquares := 0.0
	for _, val := range v {
		sumSquares += val * val
	}
	if sumSquares == 0.0 {
		return
	}
	magnitude := math.Sqrt(sumSquares)
	for k := range v {
		v[k] /= magnitude
	}
}

func extractCleanWords(s string) []string {
	var words []string
	var cur strings.Builder

	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
		} else {
			if cur.Len() > 0 {
				words = append(words, cur.String())
			}
			cur.Reset()
		}
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words
}

// canonicalStem maps common programming synonyms into canonical semantic root concepts.
func canonicalStem(w string) string {
	synonymMap := map[string]string{
		"flip":      "reverse",
		"invert":    "reverse",
		"backwards": "reverse",
		"reversing": "reverse",
		"text":      "string",
		"str":       "string",
		"strings":   "string",
		"func":      "function",
		"method":    "function",
		"def":       "function",
		"connect":   "attach",
		"link":      "attach",
		"create":    "build",
		"make":      "build",
		"fetch":     "get",
		"retrieve":  "get",
		"golang":    "go",
		"py":        "python",
	}

	if stem, ok := synonymMap[w]; ok {
		return stem
	}
	return w
}
