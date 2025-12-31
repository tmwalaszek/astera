package handler

import (
	"astera"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

const (
	sumDbPath = "/sumdb/sum.golang.org/supported"
)

type Handler struct {
	cache astera.GoProxyService
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, 0}
}

func (l *loggingResponseWriter) WriteHeader(code int) {
	l.statusCode = code
	l.ResponseWriter.WriteHeader(code)
}

func LoggerMiddlerware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		before := time.Now()
		lw := newLoggingResponseWriter(w)
		next.ServeHTTP(lw, r)
		elapsed := time.Since(before)
		slog.Info("Received HTTP Request", "Method", r.Method,
			"Path", r.URL.Path,
			"Status", lw.statusCode,
			"RemoteAddr", r.RemoteAddr,
			"RequestURI", r.RequestURI,
			"Elapsed", elapsed)
	})

}

func NewHandler(cache astera.GoProxyService) *Handler {
	return &Handler{cache: cache}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// we do not support sumdb atm so the requests should go to the official sumdb
	if r.URL.Path == sumDbPath {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	resp, err := h.cache.Query(r.Context(), r.URL.Path)
	if err != nil {
		if errors.Is(err, astera.ErrModuleNotFound) {
			http.Error(w, astera.ErrModuleNotFound.Error(), http.StatusNotFound)
			return
		}

		slog.Error("query failed", "path", r.URL.Path, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(resp)
	if err != nil {
		slog.Error("failed to write response body", "path", r.URL.Path, "err", err)
	}
}
