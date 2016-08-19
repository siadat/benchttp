package benchttp_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/siadat/benchttp/benchttp"
)

func ExampleBenchttp() {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}))
	defer s.Close()

	req, err := http.NewRequest("GET", s.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	b := &benchttp.Benchttp{
		Concurrency: 10,
		Request:     req,
	}
	report := b.SendNumber(10)
	fmt.Println(report.RequestCount)
	// Output:
	// 10
}

func TestBenchttpDuration(t *testing.T) {
	count := uint64(0)
	h := func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("hello"))
		atomic.AddUint64(&count, 1)
	}
	s := httptest.NewServer(http.HandlerFunc(h))
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)

	d := 2 * time.Second
	b := &benchttp.Benchttp{Concurrency: 1000, Request: req}
	r := b.SendDuration(d)
	okDiff := 100 * time.Millisecond
	if r.Duration-d > okDiff {
		t.Errorf("Expected %+v < d < %+v, it lasted for %+v", d, d+okDiff, r.Duration)
	}
}

func TestBenchttpNumber(t *testing.T) {
	count := uint64(0)
	h := func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("hello"))
		atomic.AddUint64(&count, 1)
	}
	s := httptest.NewServer(http.HandlerFunc(h))
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)

	n := uint64(20)
	b := &benchttp.Benchttp{Concurrency: 10, Request: req}
	b.SendNumber(n)
	if count != n {
		t.Errorf("Expected %d requests, received %d", n, count)
	}
}
