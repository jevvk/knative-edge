package main

import (
	"fmt"
	"function/pkg/worker"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
)

const DEFAULT_BLOCK_SIZE = 4096
const DEFAULT_BYTES = 32 * 1024 * 1024 // 32MB

// Handle an HTTP Request.
func Handle(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" || req.URL.Path != "/" {
		http.Error(res, fmt.Sprintf("not found: %s %s", req.Method, req.URL.Path), http.StatusNotFound)
		return
	}

	var err error
	var blockSize int
	var bytes int

	queryParams := req.URL.Query()
	blockSizeStr := queryParams.Get("block_size")
	bytesStr := queryParams.Get("bytes")

	if bytesStr == "" {
		bytes = DEFAULT_BYTES
	} else if bytes, err = strconv.Atoi(bytesStr); err != nil {
		http.Error(res, fmt.Sprintf("bad request: invalid bytes %s", bytesStr), http.StatusBadRequest)
		return
	}

	if blockSizeStr == "" {
		blockSize = DEFAULT_BLOCK_SIZE
	} else if blockSize, err = strconv.Atoi(blockSizeStr); err != nil {
		http.Error(res, fmt.Sprintf("bad request: invalid bytes %s", blockSizeStr), http.StatusBadRequest)
		return
	}

	if err := randomIO(bytes, blockSize); err != nil {
		http.Error(res, fmt.Sprintf("internal server error: %s", err), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(fmt.Sprintf("ok: bytes=%d block_size=%d", bytes, blockSize)))
}

func randomIO(bytes, blockSize int) error {
	f, err := os.CreateTemp("", "tmp-randomio-")

	if f != nil {
		defer os.Remove(f.Name())
		defer f.Close()
	}

	if err != nil {
		return fmt.Errorf("cannot create temporary file: %w", err)
	}

	block := make([]byte, blockSize)
	rand.Read(block)

	index := 0

	for index < bytes {
		written, err := f.Write(block)

		if err != nil {
			return err
		}

		index += written
	}

	f.Seek(0, 0)

	for {
		_, err := f.Read(block)

		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	w := worker.Worker{
		Name:    "random-io",
		Handler: Handle,
	}

	w.Main()
}
