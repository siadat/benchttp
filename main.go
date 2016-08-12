package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	start time.Time

	reqErrCount, reqDoneCount, reqSentCount int

	// requests is used to queue requests
	requests = make(chan *http.Request, *flagConcurrency-1)

	// runnings is used to limit the number of concurrently running requests to
	// the max specified by flagConcurrency.
	runnings = make(chan bool, *flagConcurrency-1)

	lock             sync.RWMutex
	statusCodeCounts = make(map[int]int)

	flagURL         = ""
	flagNumber      = flag.Int("n", 0, "max number of requests")
	flagDuration    = flag.Duration("d", 0, "max benchmark duration")
	flagConcurrency = flag.Int("c", 1, "max concurrent requests")
)

func main() {
	flag.Usage = func() {
		fmt.Println(`Usage: benchttp [-n 1000] [-d 1s] [-c 1] http[s]://host[:port]/path`)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *flagDuration == 0 && *flagNumber == 0 {
		*flagDuration = time.Second
	}

	if flag.NArg() < 1 {
		log.Fatal("Specify a URL")
	}

	flagURL = flag.Args()[0]

	fmt.Printf("Running with -d %v -c %v -n %d\n", *flagDuration, *flagConcurrency, *flagNumber)

	start = time.Now()

	go queueRequests()
	sendRequests()

	duration := time.Since(start)
	resTotal := 0
	for i := range statusCodeCounts {
		resTotal += statusCodeCounts[i]
	}
	fmt.Printf(" Duration: %0.3fs\n", duration.Seconds())
	fmt.Printf(" Requests: %d (%0.1f/s)\n", reqDoneCount, float64(reqDoneCount)/duration.Seconds())
	fmt.Printf("   Errors: %d (%%%0.0f)\n", reqErrCount, 100*float32(reqErrCount)/float32(reqDoneCount))
	fmt.Printf("Responses: %d (%0.1f/s)\n", resTotal, float64(resTotal)/duration.Seconds())
	for code, count := range statusCodeCounts {
		fmt.Printf("      %d: %d (%%%0.1f)\n", code, count, 100*float32(count)/float32(resTotal))
	}
}

// queueRequests creates requests and sends them to the requests channel. It
// stops creating requests after flagDuration or flagNumber is reached.
func queueRequests() {
	var sentReq *http.Request
	var err error
	for {
		if (*flagDuration != 0 && time.Since(start) >= *flagDuration) || (*flagNumber != 0 && reqSentCount >= *flagNumber) {
			close(requests)
			break
		}

		sentReq, err = http.NewRequest("GET", flagURL, nil)
		if err != nil {
			log.Fatal("NewRequest:", err)
		}

		requests <- sentReq
		reqSentCount++
	}
}

// sendRequests receives queued requests channel and sends them. The max number
// or running requests is limied by flagConcurrency.
func sendRequests() {
	for req := range requests {
		sinceStart := time.Since(start)
		if *flagDuration > 0 && sinceStart >= *flagDuration {
			break
		}

		go func() {
			if *flagDuration > 0 {
				d := *flagDuration - sinceStart
				if d < 100*time.Millisecond {
					d = 100 * time.Millisecond
				}
				http.DefaultClient.Timeout = d
			}
			reqSentCount++
			res, err := http.DefaultClient.Do(req)
			res.Body.Close()
			<-runnings

			reqDoneCount++
			if err != nil {
				reqErrCount++
			} else {
				lock.Lock()
				statusCodeCounts[res.StatusCode]++
				lock.Unlock()
			}
		}()
		runnings <- true
	}
}
