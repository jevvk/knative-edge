package event

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func Encode(encoding EncodingType, obj any) ([]byte, error) {
	if obj == nil {
		return nil, errors.New("object cannot be null")
	}

	switch encoding {
	case JsonEncoding:
		return json.Marshal(obj)

	case CompressedEncodingV1:
		jBuf, err := json.Marshal(obj)

		if err != nil {
			return nil, fmt.Errorf("encoding error: %s", err)
		}

		b64Buf := make([]byte, base64.StdEncoding.EncodedLen(len(jBuf)))
		base64.StdEncoding.Encode(jBuf, b64Buf)

		var buf bytes.Buffer

		w := zlib.NewWriter(&buf)
		w.Write(b64Buf)
		w.Close()

		return buf.Bytes(), nil
	}

	return nil, fmt.Errorf("unknown encoding type %s", encoding)
}

func Decode(encoding EncodingType, data []byte, obj any) error {
	switch encoding {
	case JsonEncoding:
		return json.Unmarshal(data, obj)

	case CompressedEncodingV1:
		r, err := zlib.NewReader(bytes.NewReader(data))

		if err != nil {
			return fmt.Errorf("error reading zlib data: %s", err)
		}

		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		r.Close()

		if err != nil {
			return fmt.Errorf("error copying zlib data: %s", err)
		}

		b64Buf := buf.Bytes()
		jBuf := make([]byte, base64.StdEncoding.DecodedLen(len(b64Buf)))

		_, err = base64.StdEncoding.Decode(jBuf, b64Buf)

		if err != nil {
			return fmt.Errorf("error decoding base64 data: %s", err)
		}

		return json.Unmarshal(jBuf, obj)
	}

	return fmt.Errorf("unknown encoding type %s", encoding)
}