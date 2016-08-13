package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	start time.Time

	reqErrCount  int
	reqDoneCount int
	reqSentCount int

	// requests is used to queue requests
	requests chan *http.Request

	// runnings is used to limit the number of concurrently running requests to
	// the max specified by flagConcurrency.
	runnings chan bool

	lock             sync.RWMutex
	statusCodeCounts = make(map[int]int)
	errorsCount      = make(map[string]int)

	flagURL *url.URL

	flagHead        = flag.Bool("i", false, "do HEAD requests instead of GET")
	flagNumber      = flag.Int("n", 0, "max number of requests")
	flagVerbose     = flag.Bool("v", false, "print errors and their frequencies")
	flagDuration    = flag.Duration("d", 0, "max benchmark duration")
	flagConcurrency = flag.Int("c", 1, "max concurrent requests")
)

// queueRequests creates requests and sends them to the requests channel. It
// stops creating requests after flagDuration or flagNumber is reached.
func queueRequests() {
	var sentReq *http.Request
	var err error
	method := "GET"
	if *flagHead {
		method = "HEAD"
	}
	for {
		if (*flagDuration != 0 && time.Since(start) >= *flagDuration) ||
			(*flagNumber != 0 && reqSentCount >= *flagNumber) {
			close(requests)
			break
		}

		sentReq, err = http.NewRequest(method, flagURL.String(), nil)

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
			<-runnings

			reqDoneCount++
			if err != nil {
				reqErrCount++
				if *flagVerbose {
					lock.Lock()
					errorsCount[err.Error()]++
					lock.Unlock()
				}
			} else {
				ioutil.ReadAll(res.Body)
				res.Body.Close()
				lock.Lock()
				statusCodeCounts[res.StatusCode]++
				lock.Unlock()
			}
		}()
		runnings <- true
	}
}

func main() {
	flag.Usage = func() {
		fmt.Println(`Usage: benchttp [-n 1000] [-d 1s] [-c 1] [-v] [-i] http[s]://host[:port]/path`)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *flagDuration == 0 && *flagNumber == 0 {
		*flagDuration = time.Second
	}

	if flag.NArg() < 1 {
		log.Fatal("Specify a URL")
	}

	u := flag.Args()[0]
	if !strings.HasPrefix(u, "http") {
		u = "http://" + u
	}

	var err error
	flagURL, err = url.Parse(u)
	if err != nil {
		log.Fatal(err)
	}

	// requests is used to queue requests
	requests = make(chan *http.Request, *flagConcurrency-1)

	// runnings is used to limit the number of concurrently running requests to
	// the max specified by flagConcurrency.
	runnings = make(chan bool, *flagConcurrency-1)

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
	if *flagVerbose {
		for err, count := range errorsCount {
			fmt.Printf("\n%d times:\n%s\n", count, err)
		}
	}
}
