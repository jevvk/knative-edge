package main

import (
	"encoding/json"
	"fmt"
	"function/pkg/worker"
	"io"
	"net/http"
)

type Matrix struct {
	Data    [][]float32 `json:",inline"`
	Rows    int
	Columns int
}

func (m *Matrix) MarshalJSON() ([]byte, error) {
	if m.Data == nil {
		return []byte("null"), nil
	}

	return json.Marshal(m.Data)
}

func (m *Matrix) UnmarshalJSON(b []byte) error {
	var data [][]float32

	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	rows := len(data)

	if rows == 0 {
		return fmt.Errorf("empty data")
	}

	columns := len(data[0])

	for _, row := range data {
		if len(row) == columns {
			continue
		}

		return fmt.Errorf("data is not a matrix (found rows of different lengths: %d and %d)", columns, len(row))
	}

	m.Data = data
	m.Rows = rows
	m.Columns = columns

	return nil
}

func (mA *Matrix) Multiply(mB *Matrix) (*Matrix, error) {
	if mB == nil {
		return nil, fmt.Errorf("cannot multiply with no matrix")
	}

	if mA.Columns != mB.Rows {
		return nil, fmt.Errorf("first matrix column count and second matrix row count don't match (%d vs %d)", mA.Columns, mB.Rows)
	}

	result := make([][]float32, 0, mA.Rows)

	for i := 0; i < mA.Rows; i++ {
		row := make([]float32, 0, mB.Columns)

		for j := 0; j < mB.Columns; j++ {
			var sum float32 = 0.0

			for k := 0; k < mA.Columns; k++ {
				sum += mA.Data[i][k] * mB.Data[k][j]
			}

			row = append(row, sum)
		}

		result = append(result, row)
	}

	return &Matrix{
		Data:    result,
		Rows:    mA.Rows,
		Columns: mB.Columns,
	}, nil
}

// Handle an HTTP Request.
func Handle(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" || req.URL.Path != "/" {
		http.Error(res, fmt.Sprintf("not found: %s %s", req.Method, req.URL.Path), http.StatusNotFound)
		return
	}

	if contentType := req.Header.Get("Content-Type"); contentType != "text/plain" && contentType != "application/x-jsonlines" {
		http.Error(res, fmt.Sprintf("unsupported content type: %s (supports text/plain, application/x-jsonlines)", contentType), http.StatusNotFound)
		return
	}

	dec := json.NewDecoder(req.Body)

	if dec == nil {
		http.Error(res, "internal server error: could not create decoder", http.StatusInternalServerError)
		return
	}

	matrixA := Matrix{}
	matrixB := Matrix{}

	if err := dec.Decode(&matrixA); err != nil {
		if err == io.EOF {
			http.Error(res, "bad request: missing first matrix", http.StatusInternalServerError)
		} else {
			http.Error(res, fmt.Sprintf("bad request: could not decode first matrix: %s", err), http.StatusInternalServerError)
		}

		return
	}

	if err := dec.Decode(&matrixB); err != nil {
		if err == io.EOF {
			http.Error(res, "bad request: missing second matrix", http.StatusInternalServerError)
		} else {
			http.Error(res, fmt.Sprintf("bad request: could not decode second matrix: %s", err), http.StatusInternalServerError)
		}

		return
	}

	var err error
	var matrixC *Matrix

	for i := 0; i < 3; i++ {
		matrixC, err = matrixA.Multiply(&matrixB)

		if err != nil {
			http.Error(res, fmt.Sprintf("bad request: %s", err), http.StatusBadRequest)
			return
		}
	}

	result, err := json.Marshal(&matrixC)

	if err != nil {
		http.Error(res, fmt.Sprintf("internal server error: could not send result: %s", err), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(result)
}

func main() {
	w := worker.Worker{
		Name:    "matrix-multiply",
		Handler: Handle,
	}

	w.Main()
}
