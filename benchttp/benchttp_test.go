package benchttp_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/siadat/benchttp/benchttp"
)

func TestBenchttpDuration(t *testing.T) {
	count := uint64(0)
	h := func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("hello"))
		atomic.AddUint64(&count, 1)
	}
	s := httptest.NewServer(http.HandlerFunc(h))
	defer s.Close()

	req, _ := http.NewRequest("GET", s.URL, nil)

	d := 5 * time.Second
	b := &benchttp.Benchttp{Concurrency: 1000, Request: req}
	b.SendDuration(d)
	okDiff := 50 * time.Millisecond
	if b.Elapsed()-d > okDiff {
		t.Errorf("Expected %+v < d < %+v, it lasted for %+v", d, d+okDiff, b.Elapsed())
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
