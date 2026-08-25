package auth

import (
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"net/http"
)

func (s Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _, err := s.Authenticate(r.Context(), TokenFromRequest(r))
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func Require(next http.Handler, roles ...domain.Role) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok || domain.RequireRole(user.Role, roles...) != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
