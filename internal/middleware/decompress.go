package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type decompressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newDecompressReader(r io.ReadCloser) (*decompressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &decompressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c decompressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c decompressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}

	return c.zr.Close()
}

func DecompressBodyReader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		contentEncoding := req.Header.Get("Content-Encoding")
		if strings.Contains(contentEncoding, "gzip") {
			decompressReader, err := newDecompressReader(req.Body)
			if err != nil {
				http.Error(resp, "cannot decompress request body", http.StatusBadRequest)
				return
			}

			defer decompressReader.Close()

			req.Body = decompressReader
		}

		next.ServeHTTP(resp, req)
	})
}
