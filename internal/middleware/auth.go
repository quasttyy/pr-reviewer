package middleware

import (
	"net/http"

	"github.com/quasttyy/pr-reviewer/internal/config"
)

type Auth struct {
	admin string
	user  string
}

func NewAuth(cfg *config.Config) *Auth {
	return &Auth{
		admin: cfg.Security.AdminToken,
		user:  cfg.Security.UserToken,
	}
}

// UserOrAdmin позволяет доступ при валидном пользовательском или админском токене.
func (a *Auth) UserOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.check(r) {
			next.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
}

// AdminOnly позволяет доступ только при валидном админском токене.
func (a *Auth) AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token := bearer(r); token == a.admin && token != "" {
			next.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func (a *Auth) check(r *http.Request) bool {
	token := bearer(r)
	if token == "" {
		return false
	}
	return token == a.user || token == a.admin
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const p = "Bearer "
	if len(h) > len(p) && h[:len(p)] == p {
		return h[len(p):]
	}
	return ""
}
