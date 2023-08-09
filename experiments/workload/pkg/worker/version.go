package worker

import (
	"fmt"
	"net/http"
)

// the next line is changed by experiments script
const Version = "20861cb4-0"

func (w *Worker) versionHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(res, fmt.Sprintf("not found: %s %s", req.Method, req.URL.Path), http.StatusNotFound)
		return
	}

	http.Error(res, Version, http.StatusOK)
}
