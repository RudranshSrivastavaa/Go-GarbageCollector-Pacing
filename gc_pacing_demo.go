// Demonstrates how GOGC changes the Go garbage collector's pacing:
// how often it runs, how long it pauses, and how much memory it's
// willing to hold onto in order to run less often.
//
// Compare pacing across GOGC values:
//   go build -o demo gc_pacing_demo.go
//   for g in 25 50 100 200 400; do GOGC=$g ./demo; done
//
// Watch the raw GC trace (this is where "mark assist" shows up):
//   GOGC=50 GODEBUG=gctrace=1 ./demo
//
package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

type node struct {
	payload [512]byte
	next    *node
}

// allocateGarbage simulates request-scoped allocations: mostly
// short-lived, with a small fraction kept alive (think cache entries).
func allocateGarbage(iterations int) {
	var live *node
	for i := 0; i < iterations; i++ {
		n := &node{next: live}
		if i%500 == 0 {
			live = n
		}
	}
	runtime.KeepAlive(live)
}

func main() {
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	const workers = 8
	const perWorker = 1_000_000

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allocateGarbage(perWorker)
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	runtime.ReadMemStats(&after)

	fmt.Printf(
		"GOGC=%-5s  wall=%-9v  gc_cycles=%-5d  stw_pause_total=%-10v  peak_heap=%.1fMB\n",
		os.Getenv("GOGC"),
		elapsed.Round(time.Millisecond),
		after.NumGC-before.NumGC,
		time.Duration(after.PauseTotalNs-before.PauseTotalNs),
		float64(after.HeapSys)/1024/1024,
	)
}