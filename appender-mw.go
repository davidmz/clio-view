package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
)

func appender(text string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)
			for k, v := range rec.Header() {
				w.Header()[k] = v
			}
			if strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
				rec.Body.WriteString(text)
				w.Header().Set("Content-Length", strconv.Itoa(rec.Body.Len()))
			}
			w.WriteHeader(rec.Code)
			rec.Body.WriteTo(w)
		})
	}
}
