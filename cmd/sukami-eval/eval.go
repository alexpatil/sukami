package main

import (
	"runtime"
	"slices"
	"strconv"
	"sukami/pkg/cache"
	"sync"
	"time"
)

// outcome counts what the cache did with the second text of each pair.
type outcome struct {
	samePairs int // pairs labelled as sharing a meaning
	diffPairs int // pairs labelled as different in meaning
	trueHits  int // same-meaning pair, cache served the pair's own first text
	falseHits int // different-meaning pair, cache served the pair's own first text
	otherHits int // cache served another pair's text; the data set has no label for that combination
	exactHits int // the stored text and the looked-up text are the same string once normalised

	lookups []time.Duration
}

// medianLookup is the median wall time of the similarity lookups that were timed one at a time.
func (o outcome) medianLookup() time.Duration {
	if len(o.lookups) == 0 {
		return 0
	}
	sorted := slices.Clone(o.lookups)
	slices.Sort(sorted)
	return sorted[len(sorted)/2]
}

// evalIsolated stores the first text of each pair in a namespace of its own and looks up the second,
// so each lookup has exactly one candidate.
func evalIsolated(pairs []pair, threshold float64) outcome {
	c := cache.NewMemoryCacheWithCapacity(time.Hour, len(pairs)+1)
	namespaces := make([]string, len(pairs))
	for i, p := range pairs {
		namespaces[i] = "pair-" + strconv.Itoa(i)
		c.SetSemantic(cache.ComputeHash(namespaces[i], p.first), p.first, []byte(p.first), namespaces[i])
	}
	return lookupAll(c, pairs, threshold, func(i int) string { return namespaces[i] })
}

// evalShared stores every first text in one namespace before looking up the second texts,
// so each lookup competes against all stored prompts, as it would in a busy cache.
func evalShared(pairs []pair, threshold float64) outcome {
	const namespace = "shared"
	c := cache.NewMemoryCacheWithCapacity(time.Hour, len(pairs)+1)
	for _, p := range pairs {
		c.SetSemantic(cache.ComputeHash(namespace, p.first), p.first, []byte(p.first), namespace)
	}
	return lookupAll(c, pairs, threshold, func(int) string { return namespace })
}

// latencySample is how many lookups are timed one at a time before the parallel pass,
// so the reported lookup time is not inflated by the other workers.
const latencySample = 100

func lookupAll(c *cache.MemoryCache, pairs []pair, threshold float64, namespaceOf func(int) string) outcome {
	var total outcome
	for i := 0; i < len(pairs) && len(total.lookups) < latencySample; i++ {
		// A text that is stored verbatim returns from the hash lookup and never reaches the similarity scan.
		if _, stored := c.Get(cache.ComputeHash(namespaceOf(i), pairs[i].second)); stored {
			continue
		}
		start := time.Now()
		c.GetSemantic(namespaceOf(i), pairs[i].second, threshold)
		total.lookups = append(total.lookups, time.Since(start))
	}

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	jobs := make(chan int)
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var local outcome
			for i := range jobs {
				local.add(pairs[i], c, namespaceOf(i), threshold)
			}
			mu.Lock()
			total.merge(local)
			mu.Unlock()
		}()
	}
	for i := range pairs {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return total
}

func (o *outcome) add(p pair, c *cache.MemoryCache, namespace string, threshold float64) {
	if p.same {
		o.samePairs++
	} else {
		o.diffPairs++
	}
	served, hit, _ := c.GetSemantic(namespace, p.second, threshold)
	if !hit {
		return
	}
	switch {
	case cache.NormalizePrompt(string(served)) == cache.NormalizePrompt(p.second):
		o.exactHits++
	case string(served) != p.first:
		o.otherHits++
	case p.same:
		o.trueHits++
	default:
		o.falseHits++
	}
}

func (o *outcome) merge(other outcome) {
	o.samePairs += other.samePairs
	o.diffPairs += other.diffPairs
	o.trueHits += other.trueHits
	o.falseHits += other.falseHits
	o.otherHits += other.otherHits
	o.exactHits += other.exactHits
}
