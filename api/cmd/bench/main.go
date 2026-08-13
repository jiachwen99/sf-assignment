// Command bench measures list latency over HTTP, which is what somebody using
// the application actually waits for: the query, the encoding and the transfer,
// not just the time Postgres reports.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"time"
)

type probe struct {
	name  string
	query string
}

// The default list, each sort key, and the combinations the plan calls the
// worst case: a filter that matches a small fraction of rows, sorted.
var probes = []probe{
	{"default list", ""},
	{"sorted by due", "sort=due&dir=asc"},
	{"sorted by name", "sort=name&dir=asc"},
	{"sorted by priority", "sort=priority&dir=asc"},
	{"sorted by status", "sort=status&dir=asc"},
	{"filtered by status", "status=in_progress"},
	{"overdue", "overdue=true"},
	{"blocked", "blocked=true"},
	{"blocked, sorted by due", "blocked=true&sort=due&dir=asc"},
	{"name search", "name=ledger"},
	{"counts", ""},
}

func main() {
	base := flag.String("url", "http://localhost:8080", "where the API is")
	runs := flag.Int("runs", 50, "requests per probe, after a warm-up")
	flag.Parse()

	if err := run(*base, *runs); err != nil {
		log.Fatal(err)
	}
}

func run(base string, runs int) error {
	client := &http.Client{Timeout: 30 * time.Second}

	fmt.Printf("%-24s %8s %8s %8s\n", "query", "p50", "p95", "max")
	fmt.Println("------------------------ -------- -------- --------")

	for _, p := range probes {
		url := base + "/api/todos?" + p.query
		if p.name == "counts" {
			url = base + "/api/todos/counts"
		}

		// One request that is not measured, so the first connection and the
		// first plan are not counted as latency.
		if err := once(client, url); err != nil {
			return err
		}

		took := make([]time.Duration, runs)
		for i := range took {
			start := time.Now()
			if err := once(client, url); err != nil {
				return err
			}
			took[i] = time.Since(start)
		}
		sort.Slice(took, func(i, j int) bool { return took[i] < took[j] })

		fmt.Printf("%-24s %8s %8s %8s\n", p.name,
			ms(took[len(took)/2]), ms(took[percentile(len(took), 95)]), ms(took[len(took)-1]))
	}
	return nil
}

// The lowest index at or above the percentile, which for fifty runs makes p95
// the second slowest rather than a value nothing measured.
func percentile(n, p int) int {
	i := (n*p + 99) / 100
	if i >= n {
		return n - 1
	}
	return i
}

func ms(d time.Duration) string {
	return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
}

func once(client *http.Client, url string) error {
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Read the body: a measurement that stops at the headers is not measuring
	// the response.
	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "%s returned %d\n", url, res.StatusCode)
		return fmt.Errorf("unexpected status %d", res.StatusCode)
	}
	return nil
}
