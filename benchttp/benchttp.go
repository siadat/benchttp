package benchttp

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Benchttp holds configurations for a benchmark.
type Benchttp struct {
	// Concurrency indicates the max number of concurrent requests.
	Concurrency int

	// Request specifies the request to used in benchmarking.
	Request *http.Request

	// start is set at the beginning of the benchmark.
	start time.Time

	// end is set after the benchmark is complete.
	end time.Time

	duration time.Duration
	number   uint64

	lockCodes   sync.RWMutex
	statusCodes map[int]int

	lockErr  sync.RWMutex
	errCount map[string]int

	// idleClients limits the number of concurrently running requests to the
	// number specified by Concurrency.
	idleClients chan *http.Client

	// wg for running requests
	wg sync.WaitGroup

	// reqErrCount indicates the number of errors.
	reqErrCount uint64

	// reqDoneCount indicates the number of received responses in the
	// benchmarking time.
	reqDoneCount uint64
}

// Report is the result of a benchmark.
type Report struct {
	Duration     time.Duration
	RequestCount uint64
	StatusCodes  map[int]int
	Errors       map[string]int
}

// SendNumber starts benchmarking for n requests.
func (b *Benchttp) SendNumber(n uint64) *Report {
	b.number = n
	return b.do()
}

// SendNumber starts benchmarking for the d duration.
func (b *Benchttp) SendDuration(d time.Duration) *Report {
	b.duration = d
	return b.do()
}

func (b *Benchttp) do() *Report {
	b.statusCodes = make(map[int]int)
	b.errCount = make(map[string]int)
	b.createClients()
	b.start = time.Now()
	b.sendRequests()
	b.end = time.Now()
	return &Report{
		Duration:     b.elapsed(),
		RequestCount: b.reqDoneCount,
		StatusCodes:  b.statusCodes,
		Errors:       b.errCount,
	}
}

// elapsed is the total benchmarking duration.
func (b *Benchttp) elapsed() time.Duration {
	return b.end.Sub(b.start)
}

// Print formats the benchmarking report.
func (r *Report) Print() {
	resTotal := 0
	for i := range r.StatusCodes {
		resTotal += r.StatusCodes[i]
	}

	errTotal := 0
	for i := range r.Errors {
		errTotal += r.Errors[i]
	}

	fmt.Printf(" Duration: %0.3fs\n", r.Duration.Seconds())
	fmt.Printf(" Requests: %d (%0.1f/s)\n", r.RequestCount, float64(r.RequestCount)/r.Duration.Seconds())

	if errTotal > 0 {
		fmt.Printf("   Errors: %d\n", errTotal)
	}

	fmt.Printf("Responses: %d (%0.1f/s)\n", resTotal, float64(resTotal)/r.Duration.Seconds())
	for code, count := range r.StatusCodes {
		fmt.Printf("    [%d]: %d\n", code, count)
	}
	for err, count := range r.Errors {
		fmt.Printf("\n%d times:\n%s\n", count, err)
	}
}

func (b *Benchttp) sendOne(c *http.Client) {
	defer func() {
		// send client back to idleClients.
		b.idleClients <- c
		b.wg.Done()
	}()

	if b.duration > 0 {
		c.Timeout = b.duration - time.Since(b.start)
	}

	res, err := c.Do(b.Request)
	if err == nil {
		io.Copy(ioutil.Discard, res.Body)
		res.Body.Close()
	}

	if b.isDurationOver() {
		// ignore this response because it was received too late.
		return
	}

	atomic.AddUint64(&b.reqDoneCount, 1)

	if err != nil {
		atomic.AddUint64(&b.reqErrCount, 1)
		b.lockErr.Lock()
		b.errCount[err.Error()]++
		b.lockErr.Unlock()
		return
	}

	b.lockCodes.Lock()
	b.statusCodes[res.StatusCode]++
	b.lockCodes.Unlock()
}

// sendRequests receives idle clients and sends a request with each one until
// either benchmark duration is over or number requests are sent.
// sendRequests returns after all requests are completed.
func (b *Benchttp) sendRequests() {
	defer b.wg.Wait()
	for n := uint64(0); (b.number == 0 || b.number > n) && !b.isDurationOver(); n++ {
		b.wg.Add(1)
		go b.sendOne(<-b.idleClients)
	}
}

func (b *Benchttp) isDurationOver() bool {
	return b.duration != 0 && time.Since(b.start) > b.duration
}

// createClients creates Concurrency idle clients.
func (b *Benchttp) createClients() {
	b.idleClients = make(chan *http.Client, b.Concurrency)
	for i := 0; i < b.Concurrency; i++ {
		b.idleClients <- &http.Client{
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
