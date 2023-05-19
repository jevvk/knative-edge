package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

var (
	remoteURL      *url.URL
	remoteProxyURL *url.URL
	remoteHost     string
	timeout        time.Duration = time.Second * 30
	client         *http.Client
)

const (
	defaultAddr = ":8080"
)

var dropHeaders = []string{
	"Transfer-Encoding",
	"x-envoy-upstream-service-time",
}

func handler(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()

	headers.Set("x-knative-edge-proxy-host", remoteHost)
	headers.Set("x-knative-edge-proxy", "true")
	headers.Set("x-knative-edge-proxy-url", "")

	if remoteURL == nil {
		w.Header().Add("Content-Type", "text/plain")
		http.Error(w, "bad gateway: no remote url set", http.StatusBadGateway)

		return
	}

	headers.Set("x-knative-edge-proxy-url", remoteURL.String())

	if remoteHost == "" {
		w.Header().Add("Content-Type", "text/plain")
		http.Error(w, "bad gateway: no remote host set", http.StatusBadGateway)

		return
	}

	url := *remoteURL
	url.Path = r.URL.Path
	url.RawQuery = r.URL.RawQuery
	url.Fragment = r.URL.Fragment

	body, err := io.ReadAll(r.Body)

	if err != nil {
		w.Header().Add("Content-Type", "text/plain")
		http.Error(w, fmt.Sprintf("failed to read body: %s", err), http.StatusBadRequest)

		return
	}

	remoteReq := &http.Request{
		Method:        r.Method,
		URL:           &url,
		Header:        r.Header,
		Body:          io.NopCloser(bytes.NewBuffer(body)),
		ContentLength: r.ContentLength,
		Host:          remoteHost,
	}

	remoteReq.Header.Set("Host", remoteHost)
	remoteReq.Header.Add("X-Forwarded-For", r.RemoteAddr)
	remoteReq.Header.Add("X-Forwarded-Host", r.Host)
	remoteReq.Header.Add("X-Forwarded-Proto", r.Proto)

	remoteReq.Host = remoteHost
	remoteReq = remoteReq.WithContext(r.Context())

	start := time.Now()
	res, err := client.Do(remoteReq) // this doesn't do absolute url for proxies ([GET /] vs [GET http://foo.bar/])
	end := time.Now()
	duration := end.Sub(start)

	headers.Set("x-knative-edge-proxy-duration", duration.String())

	if err != nil {
		w.Header().Add("Content-Type", "text/plain")

		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "bad gateway: couldn't proxy to remote", http.StatusBadGateway)
		} else {
			http.Error(w, fmt.Sprintf("gateway error: %s", err), http.StatusBadGateway)
		}

		return
	}

	headers.Set("x-knative-edge-proxy-upstream", res.Header.Get("x-envoy-upstream-service-time"))

	for _, header := range dropHeaders {
		res.Header.Del(header)
	}

	for header, values := range res.Header {
		for _, value := range values {
			headers.Add(header, value)
		}
	}

	w.WriteHeader(res.StatusCode)
	defer res.Body.Close()

	if res.ContentLength > 0 {
		io.CopyN(w, res.Body, res.ContentLength)
	} else {
		io.Copy(w, res.Body)
	}

	log.Printf("[%s %s] status: %d, content-length: %d, duration: %s, duration2: %s\n", r.Method, r.URL.Path, res.StatusCode, res.ContentLength, duration.String(), res.Header.Get("x-envoy-upstream-service-time"))
}

func main() {
	remoteStr := os.Getenv("REMOTE_URL")
	remoteHost = os.Getenv("REMOTE_HOST")

	if remoteStr != "" {
		var err error
		remoteURL, err = url.Parse(remoteStr)

		if err != nil {
			log.Printf("Error: cannot parse remote url: %s\n", err)
		}
	}

	remoteProxyUrl := os.Getenv("REMOTE_PROXY")

	if remoteProxyUrl != "" {
		var err error
		remoteProxyURL, err = url.Parse(remoteProxyUrl)

		if err != nil {
			log.Printf("Error: cannot read remote proxy url: %s", err)
		}
	}

	timeoutStr := os.Getenv("REMOTE_TIMEOUT")

	if timeoutStr == "" {
		log.Printf("Using default remote timeout of %s.\n", timeout.String())
	} else {
		newTimeout, err := time.ParseDuration(timeoutStr)

		if err != nil {
			log.Printf("Couldn't parse timeout env. Using default remote timeout of %s.\n", timeout.String())
		} else {
			timeout = newTimeout
			log.Printf("Set remote timeout to %s.\n", timeoutStr)
		}
	}

	addr := os.Getenv("BIND_ADDRESS")

	if addr == "" {
		addr = defaultAddr
	}

	// same as http.DefaultClient, but using custom proxy
	client = &http.Client{
		Transport: &http.Transport{
			Proxy: func(r *http.Request) (*url.URL, error) {
				return remoteProxyURL, nil
			},
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	log.Printf("Listening to x %s.\n", addr)

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
