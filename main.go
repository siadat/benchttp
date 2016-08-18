package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
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
	// startTime is set at the beginning of the benchmark.
	startTime time.Time

	// reqErrCount indicates the number of errors.
	reqErrCount uint64

	// reqDoneCount indicates the number of received responses in the
	// benchmarking time.
	reqDoneCount uint64

	// wg for running requests
	wg sync.WaitGroup

	// idleClients limits the number of concurrently running requests to the
	// number specified by flagConcurrency.
	idleClients chan *http.Client

	lockCodes   sync.RWMutex
	statusCodes = make(map[int]int)

	lockErr  sync.RWMutex
	errCount = make(map[string]int)

	flagAuth        = flag.String("u", "", "huser:pass")
	flagHead        = flag.Bool("i", false, "do HEAD requests instead of GET")
	flagNumber      = flag.Uint64("n", 0, "number of requests")
	flagHeaders     = make(flagHeaderMap)
	flagVerbose     = flag.Bool("v", false, "print errors and their frequencies")
	flagDuration    = flag.Duration("d", 0, "max benchmark duration")
	flagConcurrency = flag.Int("c", 1, "max concurrent requests")

	argURL *url.URL
)

type flagHeaderMap map[string]string

func (h flagHeaderMap) String() string {
	return "string representation"
}

func (h flagHeaderMap) Set(value string) error {
	keyVal := strings.SplitN(value, ":", 2)
	if len(keyVal) != 2 {
		return nil
	}
	h[keyVal[0]] = keyVal[1]
	return nil
}

func isDurationOver() bool {
	return *flagDuration != 0 && time.Since(startTime) > *flagDuration
}

func newRequest() (req *http.Request) {
	var err error
	method := "GET"
	if *flagHead {
		method = "HEAD"
	}

	req, err = http.NewRequest(method, argURL.String(), nil)
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
	return req
}

func send(c *http.Client) {
	defer func() {
		// send client back to idleClients.
		idleClients <- c
		wg.Done()
	}()

	if *flagDuration > 0 {
		c.Timeout = *flagDuration - time.Since(startTime)
	}

	res, err := c.Do(newRequest())

	if isDurationOver() {
		if err == nil {
			io.Copy(ioutil.Discard, res.Body)
			res.Body.Close()
		}
		// ignore this response because it was received too late.
		return
	}

	atomic.AddUint64(&reqDoneCount, 1)

	if err != nil {
		atomic.AddUint64(&reqErrCount, 1)
		if *flagVerbose {
			lockErr.Lock()
			errCount[err.Error()]++
			lockErr.Unlock()
		}
		return
	} else {
		io.Copy(ioutil.Discard, res.Body)
		res.Body.Close()
	}

	lockCodes.Lock()
	statusCodes[res.StatusCode]++
	lockCodes.Unlock()
}

// createClients creates flagConcurrency clients for sending requests.
func createClients() {
	for i := 0; i < *flagConcurrency; i++ {
		idleClients <- &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return fmt.Errorf("no redirects")
			},
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
				DisableCompression: true,
				DisableKeepAlives:  false,
			},
		}
	}
}

// sendRequests creates clients and sends requests with them. The max number of
// running clients is limited by flagConcurrency.
func sendRequests() {
	defer wg.Wait()
	for n := uint64(0); (*flagNumber == 0 || *flagNumber > n) && !isDurationOver(); n++ {
		wg.Add(1)
		go send(<-idleClients)
	}
}

func main() {
	log.SetFlags(0)

	flag.Usage = func() {
		log.Printf("Usage: %s [-n 1000] [-d 1s] [-c 1] [-v] [-i] http[s]://host[:port]/path", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Var(&flagHeaders, "H", "custom header, e.g. 'Key: Value'")
	flag.Parse()

	if *flagDuration == 0 && *flagNumber == 0 {
		// flagNumber defaults to 1000 if neither -n nor -d are given
		*flagNumber = 1000
	}

	if *flagDuration > 0 && *flagNumber > 0 {
		log.Fatal("Do not set both -d and -n.")
	}

	if flag.NArg() < 1 {
		log.Fatal("URL missing")
	}

	u := flag.Args()[0]
	if !strings.HasPrefix(u, "http") {
		u = "http://" + u
	}

	var err error
	argURL, err = url.Parse(u)
	if err != nil {
		log.Fatal(err)
	}
	log.SetFlags(log.LstdFlags)

	idleClients = make(chan *http.Client, *flagConcurrency)

	createClients()
	startTime = time.Now()
	sendRequests()
	elapsed := time.Since(startTime)

	resTotal := 0
	for i := range statusCodes {
		resTotal += statusCodes[i]
	}

	fmt.Printf(" Duration: %0.3fs\n", elapsed.Seconds())
	fmt.Printf(" Requests: %d (%0.1f/s)\n", reqDoneCount, float64(reqDoneCount)/elapsed.Seconds())
	if *flagVerbose || reqErrCount > 0 {
		fmt.Printf("   Errors: %d\n", reqErrCount)
	}
	fmt.Printf("Responses: %d (%0.1f/s)\n", resTotal, float64(resTotal)/elapsed.Seconds())
	for code, count := range statusCodes {
		fmt.Printf("    [%d]: %d\n", code, count)
	}
	if *flagVerbose {
		for err, count := range errCount {
			fmt.Printf("\n%d times:\n%s\n", count, err)
		}
	}
}
