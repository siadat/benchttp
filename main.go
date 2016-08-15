package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	start time.Time

	reqErrCount    uint64
	reqDoneCount   uint64
	reqQueuedCount uint64

	// reqQueue is used to queue requests
	reqQueue chan *http.Request

	// runnings is used to limit the number of concurrently running requests to
	// the max specified by flagConcurrency.
	runnings chan bool

	defaultClientTimeoutLock sync.RWMutex

	lockCodes        sync.RWMutex
	statusCodeCounts = make(map[int]int)

	lockErrors  sync.RWMutex
	errorsCount = make(map[string]int)

	flagURL *url.URL

	flagAuth        = flag.String("u", "", "huser:pass")
	flagHead        = flag.Bool("i", false, "do HEAD requests instead of GET")
	flagNumber      = flag.Uint64("n", 0, "max number of requests")
	flagHeaders     = make(flagHeaderMap)
	flagVerbose     = flag.Bool("v", false, "print errors and their frequencies")
	flagDuration    = flag.Duration("d", 0, "max benchmark duration")
	flagConcurrency = flag.Int("c", 1, "max concurrent requests")
)

type flagHeaderMap map[string]string

func (h *flagHeaderMap) String() string {
	return "string representation"
}

func (h *flagHeaderMap) Set(value string) error {
	keyVal := strings.SplitN(value, ":", 2)
	if len(keyVal) != 2 {
		return nil
	}
	(*h)[keyVal[0]] = keyVal[1]
	return nil
}

func isDurationOver() bool {
	return *flagDuration != 0 && time.Since(start) > *flagDuration
}

// queueRequests creates requests and sends them to the reqQueue channel. It
// stops creating requests after flagDuration or flagNumber is reached.
func queueRequests() {
	var req *http.Request
	var err error
	method := "GET"
	if *flagHead {
		method = "HEAD"
	}

	req, err = http.NewRequest(method, flagURL.String(), nil)

	if err != nil {
		log.Fatal("NewRequest:", err)
	}

	for key, value := range flagHeaders {
		req.Header.Add(key, value)
	}

	if *flagAuth != "" {
		userPass := strings.SplitN(*flagAuth, ":", 2)
		if len(userPass) == 2 {
			req.SetBasicAuth(userPass[0], userPass[1])
		}
	}

	for {
		if isDurationOver() || (*flagNumber != 0 && reqQueuedCount == *flagNumber) {
			close(reqQueue)
			break
		}
		reqQueue <- req
		reqQueuedCount++
	}
}

// sendRequests receives from reqQueue channel and sends them. The max number
// of running requests is limited by flagConcurrency. It returns after all
// requests are completed.
func sendRequests() {
	reqCompleted := make(chan bool)
	var reqCount uint64 = 0

	defer func() {
		for i := uint64(0); i < reqCount; i++ {
			<-reqCompleted
		}
	}()

	for req := range reqQueue {
		if isDurationOver() {
			return
		}

		reqCount++
		go func(req *http.Request) {
			defer func() {
				<-runnings
				reqCompleted <- true
			}()

			if *flagDuration > 0 {
				d := *flagDuration - time.Since(start)
				if d < 1000*time.Millisecond {
					d = 1000 * time.Millisecond
				}
				defaultClientTimeoutLock.Lock()
				http.DefaultClient.Timeout = d
				defaultClientTimeoutLock.Unlock()
			}

			res, err := http.DefaultClient.Do(req)

			if isDurationOver() {
				if err == nil {
					res.Body.Close()
				}
				return
			}

			atomic.AddUint64(&reqDoneCount, 1)

			if err != nil {
				atomic.AddUint64(&reqErrCount, 1)
				if *flagVerbose {
					lockErrors.Lock()
					errorsCount[err.Error()]++
					lockErrors.Unlock()
				}
				return
			}

			ioutil.ReadAll(res.Body)
			res.Body.Close()

			lockCodes.Lock()
			statusCodeCounts[res.StatusCode]++
			lockCodes.Unlock()
		}(req)

		runnings <- true
	}
}

func main() {
	log.SetFlags(0)

	flag.Usage = func() {
		log.Printf("Usage: %s [-n 1000] [-d 1s] [-c 1] [-v] [-i] http[s]://host[:port]/path", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Var(&flagHeaders, "H", "custom header separated by a colon, e.g. 'Key: Value'")
	flag.Parse()

	if *flagDuration == 0 && *flagNumber == 0 {
		// default to -n 1000 if neither -n nor -d are given
		*flagNumber = 1000
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
	log.SetFlags(log.LstdFlags)

	reqQueue = make(chan *http.Request, 1000)
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
	fmt.Printf(" Requests: %d (%0.1f/s)\n", atomic.LoadUint64(&reqDoneCount), float64(atomic.LoadUint64(&reqDoneCount))/duration.Seconds())
	fmt.Printf("   Errors: %d (%%%0.0f)\n", atomic.LoadUint64(&reqErrCount), 100*float32(atomic.LoadUint64(&reqErrCount))/float32(atomic.LoadUint64(&reqDoneCount)))
	fmt.Printf("Responses: %d (%0.1f/s)\n", resTotal, float64(resTotal)/duration.Seconds())
	for code, count := range statusCodeCounts {
		fmt.Printf("    [%d]: %d (%%%0.1f)\n", code, count, 100*float32(count)/float32(resTotal))
	}
	if *flagVerbose {
		for err, count := range errorsCount {
			fmt.Printf("\n%d times:\n%s\n", count, err)
		}
	}
}
