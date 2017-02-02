package benchttp_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/siadat/benchttp"
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
		t.Errorf("want %+v < d < %+v; got %+v", d, d+okDiff, r.Duration)
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
	report := b.SendNumber(n)
	if want, got := n, count; want != got {
		t.Errorf("server want %d requests; got %d", want, got)
	}
	if want, got := n, report.RequestCount; want != got {
		t.Errorf("report.RequestCount want %d requests; got %d", want, got)
	}
	if want, got := 1, len(report.StatusCodes); want != got {
		t.Errorf("len(report.StatusCodes) want %v requests; got %v", want, got)
	}
	if want, got := n, report.StatusCodes[200]; want != uint64(got) {
		t.Errorf("report.StatusCodes[200] want %v requests; got %v", want, got)
	}
}
