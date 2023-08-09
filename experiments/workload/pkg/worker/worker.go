package worker

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

type Worker struct {
	Name    string
	Handler http.HandlerFunc

	nodeName     string
	revisionName string

	inflightReqs int32
	drain        bool
	drainChan    chan int
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *Worker) workerHandlerWithDrain(res http.ResponseWriter, req *http.Request) {
	if w.drain {
		http.Error(res, "error: worker has stopped", http.StatusBadRequest)
		return
	}

	atomic.AddInt32(&w.inflightReqs, 1)

	defer func() {
		atomic.AddInt32(&w.inflightReqs, -1)

		if w.drain {
			w.drainChan <- 0
		}
	}()

	// log.Printf("New request: %s %s", req.Method, req.URL.Path)

	res.Header().Set("x-k-node-name", w.nodeName)
	w.Handler(res, req)
}

func (w *Worker) workerHandler(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Encoding") == "gzip" {
		decompressed, err := gzip.NewReader(req.Body)

		if err != nil {
			http.Error(res, fmt.Sprintf("error: bad compression: %s", err), http.StatusBadRequest)
			return
		}

		req.Body = decompressed
	}

	if req.Header.Get("Accept-Encoding") == "gzip" {
		res.Header().Set("Content-Encoding", "gzip")

		// always compresses, even for small responses
		gz := gzip.NewWriter(res)
		defer gz.Close()
		res = gzipResponseWriter{Writer: gz, ResponseWriter: res}
	}

	// log.Printf("New request: %s %s", req.Method, req.URL.Path)
	res.Header().Set("x-k-node-name", w.nodeName)
	w.Handler(res, req)
}

func (w *Worker) drainHandler(res http.ResponseWriter, req *http.Request) {
	defer func() {
		http.Error(res, "ok", http.StatusOK)
	}()

	w.drain = true
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for {
		if w.inflightReqs == 0 {
			return
		}

		select {
		case <-w.drainChan:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) init() {
	podName := os.Getenv("K_SERVICE")
	if podName == "" {
		podName = w.Name
	}

	w.nodeName = os.Getenv("K_NODE_NAME")
	if w.nodeName == "" {
		w.nodeName = fmt.Sprintf("%s-%s", podName, RandomString(6))
	}

	w.revisionName = os.Getenv("K_REVISION")

	nodes := map[string]string{
		"100.88.114.213": "edge-surface-go",
		"100.88.114.223": "edge-rpi3",
		"100.88.114.216": "edge-rpi2",
		"100.88.114.218": "edge-rpi0",
		"100.88.114.215": "edge-rpi1",
	}

	nodesMapStr := os.Getenv("K_NODES_MAP")
	nodesMapValues := strings.Split(nodesMapStr, " ")

	if len(nodesMapValues)%2 != 0 {
		log.Printf("%s: [warning] nodes map has odd number of value", w.Name)
	} else {
		value := "{nil}"

		for _, v := range nodesMapValues {
			if value == "{nil}" {
				value = v
			} else {
				nodes[v] = value
				value = "{nil}"
			}
		}
	}

	nodeName := nodes[os.Getenv("HOST_IP")]

	if nodeName != "" {
		w.nodeName = nodeName
	}

	// if w.revisionName != "" {
	// 	w.nodeName = fmt.Sprintf("%s (%s)", w.nodeName, w.revisionName)
	// }

	log.Printf("%s: Node name set to %s.", w.Name, w.nodeName)
}

func (w *Worker) Main() {
	log.Printf("%s: Starting server...", w.Name)

	w.init()

	// http.HandleFunc("//wait-for-drain", w.drainHandler)
	// http.HandleFunc("/wait-for-drain", w.drainHandler)
	// http.HandleFunc("/versionz", w.versionHandler)
	http.HandleFunc("/", w.workerHandler)

	address := os.Getenv("WORKER_ADDRESS")
	if address == "" {
		address = ":8080"
	}

	log.Printf("%s: Listening on %s.", w.Name, address)
	log.Fatal(http.ListenAndServe(address, nil))
}

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	s := make([]rune, n)

	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}

	return string(s)
}
