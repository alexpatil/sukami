package cache

// entityClusters defines mutually exclusive technical entities within the same technology domain.
var entityClusters = [][]string{
	// Containers & Orchestration
	{"docker", "podman", "kubernetes", "k8s", "nomad", "containerd"},
	// Databases & Stores
	{"postgres", "postgresql", "mysql", "sqlite", "mongodb", "mongo", "redis", "cassandra", "dynamodb", "mariadb"},
	// Programming Languages
	{"python", "golang", "rust", "java", "ruby", "php", "swift", "javascript", "typescript", "kotlin"},
	// Cloud Providers
	{"aws", "azure", "gcp", "digitalocean", "cloudflare", "heroku"},
	// Operating Systems
	{"ubuntu", "debian", "centos", "alpine", "windows", "macos", "arch"},
	// Web Frameworks
	{"react", "vue", "angular", "svelte", "nextjs", "django", "flask", "fastapi", "spring", "gin", "fiber", "express"},
}

// boilerplateWords represents generic instructional verbs and template words that inflate similarity.
var boilerplateWords = map[string]bool{
	"how": true, "do": true, "does": true, "did": true, "i": true, "me": true, "my": true,
	"a": true, "an": true, "the": true, "in": true, "on": true, "at": true, "to": true,
	"of": true, "for": true, "is": true, "are": true, "can": true, "you": true,
	"please": true, "help": true, "write": true, "give": true, "show": true, "tell": true,
	"setup": true, "configure": true, "install": true, "guide": true, "tutorial": true,
	"step": true, "steps": true, "best": true, "practices": true, "practice": true,
	"example": true, "code": true, "explain": true, "make": true, "create": true,
	"server": true, "using": true, "with": true, "simple": true, "complete": true,
}

// HasEntityConflict checks if two queries reference distinct, conflicting entities in the same domain.
// For example: "Docker on Ubuntu" vs "Kubernetes on Ubuntu" -> Conflict detected.
func HasEntityConflict(textA, textB string) bool {
	wordsA := makeWordSet(textA)
	wordsB := makeWordSet(textB)

	for _, cluster := range entityClusters {
		var entitiesInA, entitiesInB []string
		for _, entity := range cluster {
			if wordsA[entity] {
				entitiesInA = append(entitiesInA, entity)
			}
			if wordsB[entity] {
				entitiesInB = append(entitiesInB, entity)
			}
		}

		// If both queries have entities from this cluster, but their entity sets differ completely
		if len(entitiesInA) > 0 && len(entitiesInB) > 0 {
			hasOverlap := false
			for _, ea := range entitiesInA {
				for _, eb := range entitiesInB {
					if ea == eb || canonicalStem(ea) == canonicalStem(eb) {
						hasOverlap = true
						break
					}
				}
				if hasOverlap {
					break
				}
			}
			if !hasOverlap {
				return true // Conflicting entities detected in the same domain!
			}
		}
	}
	return false
}

// KeyTermJaccardOverlap calculates Jaccard similarity (|A ∩ B| / |A ∪ B|) on non-boilerplate terms.
func KeyTermJaccardOverlap(textA, textB string) float64 {
	setA := extractKeyTerms(textA)
	setB := extractKeyTerms(textB)

	if len(setA) == 0 && len(setB) == 0 {
		return 1.0
	}
	if len(setA) == 0 || len(setB) == 0 {
		return 0.0
	}

	intersection := 0
	for term := range setA {
		if setB[term] {
			intersection++
		}
	}

	union := len(setA)
	for term := range setB {
		if !setA[term] {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

func extractKeyTerms(text string) map[string]bool {
	words := extractCleanWords(text)
	terms := make(map[string]bool)
	for _, w := range words {
		if boilerplateWords[w] {
			continue
		}
		stem := canonicalStem(w)
		if len(stem) > 1 {
			terms[stem] = true
		}
	}
	return terms
}

func makeWordSet(text string) map[string]bool {
	words := extractCleanWords(text)
	set := make(map[string]bool, len(words))
	for _, w := range words {
		set[w] = true
		set[canonicalStem(w)] = true
	}
	return set
}
