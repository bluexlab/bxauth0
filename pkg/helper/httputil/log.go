package httputil

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type logWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
	body       *bytes.Buffer
}

func (w *logWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *logWriter) Write(b []byte) (int, error) {
	w.body.Write(b)

	var err error
	w.size, err = w.ResponseWriter.Write(b)
	return w.size, err
}

func newLogWriter(w http.ResponseWriter) *logWriter {
	return &logWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           bytes.NewBufferString(""),
	}
}

func (w *logWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func LogHandler(next http.Handler) http.Handler {
	dumpRequest := false
	dumpResponse := false
	if name, ok := os.LookupEnv("DEBUG_DUMP_REQUEST"); ok {
		dumpRequest, _ = strconv.ParseBool(name)
	}
	if name, ok := os.LookupEnv("DEBUG_DUMP_RESPONSE"); ok {
		dumpResponse, _ = strconv.ParseBool(name)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if dumpRequest {
			if dump, err := httputil.DumpRequest(r, true); err == nil {
				logrus.Debugf("request: %s", dump)
			}
		}

		clientIP := resolveClientIP(r)
		lw := newLogWriter(w)
		begin := time.Now()
		next.ServeHTTP(lw, r)
		elapsed := time.Since(begin).Milliseconds()

		if dumpResponse {
			logrus.Debugf("response: %s", lw.body.String())
		}
		logrus.Infof("HTTP - %s - - - \"%s %s %s\" %d %d \"%s\" \"%s\" (%dms)", clientIP, r.Method, r.URL.EscapedPath(), r.Proto, lw.statusCode, lw.size, r.Referer(), r.UserAgent(), elapsed)
	})
}

func resolveClientIP(r *http.Request) string {
	xRealIP := r.Header.Get("X-Real-IP")
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xRealIP == "" && xForwardedFor == "" {
		return r.RemoteAddr
	}
	if xForwardedFor != "" {
		parts := strings.Split(xForwardedFor, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		return parts[0]
	}
	return xRealIP
}
