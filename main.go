package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	flagAuth        = flag.String("u", "", "huser:pass")
	flagHead        = flag.Bool("i", false, "do HEAD requests instead of GET")
	flagNumber      = flag.Uint64("n", 0, "number of requests")
	flagVerbose     = flag.Bool("v", false, "print errors and their frequencies")
	flagDuration    = flag.Duration("d", 0, "max benchmark duration")
	flagConcurrency = flag.Int("c", 1, "max concurrent requests")
	flagHeaders     = make(colonSeparatedFlags)
)

type colonSeparatedFlags map[string]string

func (h colonSeparatedFlags) String() string {
	return "string representation"
}

func (h colonSeparatedFlags) Set(value string) error {
	keyVal := strings.SplitN(value, ":", 2)
	if len(keyVal) != 2 {
		return nil
	}
	h[keyVal[0]] = keyVal[1]
	return nil
}

func newRequest(url string) *http.Request {
	method := "GET"
	if *flagHead {
		method = "HEAD"
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Fatal("NewRequest:", err)
	}

	for key, value := range flagHeaders {
		req.Header.Add(key, value)
	}

	if *flagAuth != "" {
		if userPass := strings.SplitN(*flagAuth, ":", 2); len(userPass) == 2 {
			req.SetBasicAuth(userPass[0], userPass[1])
		}
	}
	return req
}

func main() {
	log.SetFlags(0)

	flag.Usage = func() {
		log.Printf("Usage: %s [-n 1000] [-d 1s] [-c 1] [-v] [-i] http[s]://host[:port]/path", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Var(&flagHeaders, "H", "custom header, e.g. 'Key: Value'")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("URL missing")
	}

	argURL, err := url.Parse(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}

	b := &Benchttp{
		Concurrency: *flagConcurrency,
		Request:     newRequest(argURL.String()),
	}

	if *flagDuration == 0 && *flagNumber == 0 {
		// assume -n 1000 if neither -n nor -d are given.
		*flagNumber = 1000
	}

	if *flagDuration > 0 && *flagNumber > 0 {
		log.Fatal("Do not set both -d and -n.")
	} else if *flagDuration > 0 {
		b.SendDuration(*flagDuration)
	} else if *flagNumber > 0 {
		b.SendNumber(*flagNumber)
	}

	b.PrintReport()
}
