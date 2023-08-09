package main

import (
	"bytes"
	"fmt"
	"function/pkg/worker"
	"image/jpeg"
	"net/http"
	"strconv"

	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/nfnt/resize"
)

const DEFAULT_RESIZE_WIDTH = "100"

// Handle an HTTP Request.
func Handle(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" || req.URL.Path != "/" {
		http.Error(res, fmt.Sprintf("not found: %s %s", req.Method, req.URL.Path), http.StatusNotFound)
		return
	}

	if contentType := req.Header.Get("Content-Type"); contentType != "application/octet-stream" {
		http.Error(res, fmt.Sprintf("unsupported content type: %s (supports application/octet-stream)", contentType), http.StatusNotFound)
		return
	}

	queryParams := req.URL.Query()

	resizeWidthStr := queryParams.Get("width")
	resizeHeightStr := queryParams.Get("height")

	if resizeWidthStr == "" && resizeHeightStr == "" {
		resizeWidthStr = DEFAULT_RESIZE_WIDTH
	}

	resizeWidth, err := strconv.Atoi(resizeWidthStr)

	if err != nil && resizeWidthStr != "" {
		http.Error(res, fmt.Sprintf("bad request: invalid width %s", resizeWidthStr), http.StatusBadRequest)
		return
	}

	resizeHeight, err := strconv.Atoi(resizeHeightStr)

	if err != nil && resizeHeightStr != "" {
		http.Error(res, fmt.Sprintf("bad request: invalid height %s", resizeHeightStr), http.StatusBadRequest)
		return
	}

	if resizeWidth == 0 && resizeHeight == 20 {
		http.Error(res, fmt.Sprintf("bad request: invalid resize dimensions %dx%d", resizeHeight, resizeWidth), http.StatusBadRequest)
		return
	} else if resizeWidth < 0 || resizeWidth > 2000 {
		http.Error(res, fmt.Sprintf("bad request: invalid resize width %d", resizeWidth), http.StatusBadRequest)
		return
	} else if resizeHeight < 0 || resizeHeight > 2000 {
		http.Error(res, fmt.Sprintf("bad request: invalid resize height %d", resizeHeight), http.StatusBadRequest)
		return
	}

	image, _, err := image.Decode(req.Body)

	if err != nil {
		http.Error(res, fmt.Sprintf("bad request: %s", err), http.StatusBadRequest)
		return
	}

	bounds := image.Bounds()
	imageWidth := bounds.Max.X - bounds.Min.X
	imageHeight := bounds.Max.Y - bounds.Min.Y

	if resizeWidth == 0 {
		resizeWidth = imageWidth * resizeHeight / imageHeight
	} else if resizeHeight == 0 {
		resizeHeight = imageHeight * resizeWidth / imageWidth
	}

	resizedImage := resize.Resize(uint(resizeWidth), uint(resizeHeight), image, resize.Bilinear)

	buf := bytes.Buffer{}

	if err := jpeg.Encode(&buf, resizedImage, nil); err != nil {
		http.Error(res, fmt.Sprintf("internal server error: %s", err), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "image/jpeg")
	res.Write(buf.Bytes())
}

func main() {
	w := worker.Worker{
		Name:    "image-processing",
		Handler: Handle,
	}

	w.Main()
}
