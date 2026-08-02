package httpapi

import (
	"net/http"
	"strings"
)

func BearerToken(r *http.Request) (string, error) {
	value := r.Header.Get("Authorization")
	if value == "" {
		return "", ErrUnauthorized
	}

	scheme, token, ok := strings.Cut(value, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", ErrUnauthorized
	}

	return token, nil
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := BearerToken(r); err != nil {
			WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "Требуется авторизация. Передайте действительный токен в заголовке Authorization.", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
