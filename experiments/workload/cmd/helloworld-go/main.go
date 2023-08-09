package main

import (
	"fmt"
	"function/pkg/worker"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

func fibonacci(n int) *big.Int {
	if n < 2 {
		return big.NewInt(int64(n))
	}

	a, b := big.NewInt(0), big.NewInt(1)

	for n--; n > 0; n-- {
		a.Add(a, b)
		a, b = b, a
	}

	return b
}

var offset int = 4

// Handle an HTTP Request.
func Handle(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" || req.URL.Path != "/" {
		http.Error(res, fmt.Sprintf("not found: %s %s", req.Method, req.URL.Path), http.StatusNotFound)
		return
	}

	n := offset

	fib := req.URL.Query().Get("FIB")
	if fib != "" {
		i, err := strconv.Atoi(fib)

		if err == nil {
			n = i
		}
	}

	n += rand.Intn(3)
	num := fibonacci(n)

	target := os.Getenv("TARGET")
	if target == "" {
		target = "World"
	}

	http.Error(res, fmt.Sprintf("Hello %s!\nfibonacci(%d) = %d\n", target, n, num), http.StatusOK)
}

func main() {
	rand.Seed(time.Now().Unix())

	fib := os.Getenv("FIB")
	if fib != "" {
		i, err := strconv.Atoi(fib)

		if err == nil {
			offset = i
		}
	}

	w := worker.Worker{
		Name:    "helloworld",
		Handler: Handle,
	}

	w.Main()
}
