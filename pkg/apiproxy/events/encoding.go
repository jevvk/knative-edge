package events

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

type EncodingType string

const (
	JsonEncoding         EncodingType = "json"
	CompressedEncodingV1 EncodingType = "compressedv1"
	DefaultEncoding      EncodingType = JsonEncoding
)

func Encode(encoding EncodingType, obj any) ([]byte, error) {
	if encoding == CompressedEncodingV1 {
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
	} else if encoding == JsonEncoding {
		return json.Marshal(obj)
	} else {
		return nil, fmt.Errorf("unknown encoding type %s", encoding)
	}
}

func Decode(encoding EncodingType, data []byte, obj any) error {
	if encoding == CompressedEncodingV1 {
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
	} else if encoding == JsonEncoding {
		return json.Unmarshal(data, obj)
	} else {
		return fmt.Errorf("unknown encoding type %s", encoding)
	}
}
