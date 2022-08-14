package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

var (
	remoteURL *url.URL
	timeout   time.Duration = time.Second * 30
)

const (
	defaultAddr = ":8080"
)

func proxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	remoteReq := r.Clone(ctx)
	remoteReq.URL.Host = remoteURL.Host

	remoteReq.Header.Del("Host")
	remoteReq.Header.Add("Host", remoteURL.Host)
	remoteReq.Header.Add("X-Forwarded-For", r.RemoteAddr)
	remoteReq.Header.Add("X-Forwarded-Host", r.Host)
	remoteReq.Header.Add("X-Forwarded-Proto", r.Proto)

	if proxyURL, _ := http.ProxyFromEnvironment(remoteReq); proxyURL != nil {
		remoteReq.URL = proxyURL
	}

	res, err := http.DefaultClient.Do(remoteReq)

	if err != nil {
		w.Header().Add("Content-Type", "text/plain")
		http.Error(w, "bad gateway: couldn't proxy to remote", http.StatusBadGateway)

		return
	}

	if err := res.Write(w); err != nil {
		fmt.Printf("[%s %s] Error: couldn't write response for: %s", r.Method, r.URL.Path, err)
	}
}

func main() {
	remoteStr := os.Getenv("REMOTE_URL")

	if remoteStr == "" {
		panic(fmt.Errorf("cannot proxy to empty remote url"))
	}

	var err error
	remoteURL, err = url.Parse(remoteStr)

	if err != nil {
		panic(fmt.Errorf("cannot parse remote url: %w", err))
	}

	timeoutStr := os.Getenv("REMOTE_TIMEOUT")

	if timeoutStr == "" {
		fmt.Printf("Using default remote timeout of %s.", timeout.String())
	} else {
		newTimeout, err := time.ParseDuration(timeoutStr)

		if err != nil {
			fmt.Printf("Couldn't parse timeout env. Using default remote timeout of %s.", timeout.String())
		} else {
			timeout = newTimeout
			fmt.Printf("Set remote timeout to %s.", timeoutStr)
		}
	}

	addr := os.Getenv("BIND_ADDRESS")

	if addr == "" {
		addr = defaultAddr
	}

	fmt.Printf("Listening to %s.", addr)

	http.HandleFunc("/", proxy)
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
