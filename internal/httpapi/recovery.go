package httpapi

import (
	"net/http"
	"runtime/debug"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				_ = debug.Stack()
				writeError(w, http.StatusInternalServerError, "internal_error", http.ErrAbortHandler, r.Header.Get("X-Request-ID"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
func Method(allowed string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != allowed {
			w.Header().Set("Allow", allowed)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}
