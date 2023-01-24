package main

import (
	"bytes"
	"crypto/sha256"
	"log"
	"net/http"
	"strings"
	"time"
)

func hashOf(vs ...[]byte) []byte {
	h := sha256.New()
	for _, v := range vs {
		h.Write(v)
	}
	return h.Sum(nil)
}

type loggingMiddleware struct {
	http.Handler

	logger *log.Logger
}

func (mw *loggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	iw := &interceptingWriter{ResponseWriter: w, code: 200}
	defer func(begin time.Time) {
		var response string
		if iw.code != http.StatusOK {
			response = " -- " + strings.TrimSpace(iw.buf.String())
		}
		mw.logger.Printf(
			"%s: %s %s -> %d (%dB) %s%s",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			iw.code,
			iw.buf.Len(),
			time.Since(begin).String(),
			response,
		)
	}(time.Now())
	mw.Handler.ServeHTTP(iw, r)
}

type interceptingWriter struct {
	http.ResponseWriter

	code int
	buf  bytes.Buffer
}

func (iw *interceptingWriter) WriteHeader(code int) {
	iw.code = code
	iw.ResponseWriter.WriteHeader(code)
}

func (iw *interceptingWriter) Write(p []byte) (int, error) {
	n, err := iw.ResponseWriter.Write(p)
	iw.buf.Write(p[:n])
	return n, err
}
