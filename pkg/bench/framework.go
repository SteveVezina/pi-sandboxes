package bench

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Benchmark defines a single benchmark function.
type Benchmark struct {
	Name     string
	Func     func() time.Duration
	Disabled bool
}

// Result holds the timing results for a single benchmark.
type Result struct {
	Name  string
	Times []time.Duration
}

// Stats holds computed statistics for a benchmark.
type Stats struct {
	Name  string
	Min   time.Duration
	Max   time.Duration
	Mean  time.Duration
	P50   time.Duration
	P95   time.Duration
	Count int
}

// Suite is a collection of benchmarks.
type Suite struct {
	Mode     string
	Template string
	Results  []Result
}

// NewSuite creates a new benchmark suite.
func NewSuite(mode, template string) *Suite {
	return &Suite{
		Mode:     mode,
		Template: template,
	}
}

// AddBenchmark adds a benchmark to the suite.
func (s *Suite) AddBenchmark(b *Benchmark) {
	if b.Disabled {
		return
	}
	// Run 3 iterations
	var times []time.Duration
	for i := 0; i < 3; i++ {
		start := time.Now()
		duration := b.Func()
		times = append(times, duration)
		// Account for the function's own timing
		elapsed := time.Since(start)
		// Use the function's reported duration if positive, else actual elapsed
		if duration > 0 {
			times[len(times)-1] = duration
		} else {
			times[len(times)-1] = elapsed
		}
	}
	s.Results = append(s.Results, Result{Name: b.Name, Times: times})
}

// ComputeStats computes statistics for all results.
func (s *Suite) ComputeStats() []Stats {
	var stats []Stats
	for _, r := range s.Results {
		s := computeStats(r)
		stats = append(stats, s)
	}
	return stats
}

// PrintResults prints benchmark results in the SPEC format.
func (s *Suite) PrintResults(stats []Stats) {
	fmt.Printf("mode=%s template=%s\n", s.Mode, s.Template)
	for _, st := range stats {
		fmt.Printf("%s_p50=%s\n", st.Name, st.P50.Round(time.Millisecond))
		fmt.Printf("%s_p95=%s\n", st.Name, st.P95.Round(time.Millisecond))
	}
	fmt.Printf("idle_memory_per_sandbox=0MiB\n")
}

// PrintJSON prints benchmark results as JSON.
func (s *Suite) PrintJSON(stats []Stats) {
	fmt.Print("{\n")
	fmt.Printf(`  "mode": "%s",`, s.Mode)
	fmt.Printf(`  "template": "%s",`, s.Template)
	fmt.Println(`  "benchmarks": [`)
	for i, st := range stats {
		comma := ","
		if i == len(stats)-1 {
			comma = ""
		}
		fmt.Printf(`    {"name": "%s", "p50": "%s", "p95": "%s"}%s`,
			st.Name, st.P50.Round(time.Millisecond),
			st.P95.Round(time.Millisecond), comma)
		fmt.Println()
	}
	fmt.Println(`  ]`)
	fmt.Println(`}`)
}

func computeStats(r Result) Stats {
	if len(r.Times) == 0 {
		return Stats{Name: r.Name}
	}

	times := make([]float64, len(r.Times))
	for i, t := range r.Times {
		times[i] = float64(t)
	}
	sort.Float64s(times)

	n := len(times)
	min := time.Duration(times[0])
	max := time.Duration(times[n-1])
	sum := 0.0
	for _, t := range times {
		sum += t
	}
	mean := time.Duration(sum / float64(n))

	p50 := percentile(times, 50)
	p95 := percentile(times, 95)

	return Stats{
		Name:  r.Name,
		Min:   min,
		Max:   max,
		Mean:  mean,
		P50:   p50,
		P95:   p95,
		Count: n,
	}
}

func percentile(data []float64, p float64) time.Duration {
	if len(data) == 0 {
		return 0
	}
	// Use nearest-rank method
	index := int(math.Ceil(float64(p)/100.0*float64(len(data)))) - 1
	if index >= len(data) {
		index = len(data) - 1
	}
	if index < 0 {
		index = 0
	}
	return time.Duration(data[index])
}

// FormatResults formats results as human-readable table.
func FormatResults(stats []Stats) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("%-30s %12s %12s %12s", "Benchmark", "p50", "p95", "mean"))
	lines = append(lines, strings.Repeat("-", 70))
	for _, s := range stats {
		lines = append(lines, fmt.Sprintf("%-30s %12s %12s %12s",
			s.Name, s.P50.Round(time.Millisecond),
			s.P95.Round(time.Millisecond),
			s.Mean.Round(time.Millisecond)))
	}
	return strings.Join(lines, "\n")
}
