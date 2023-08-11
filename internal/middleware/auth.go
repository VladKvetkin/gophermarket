package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/VladKvetkin/gophermart/internal/services/jwttoken"
)

type UserIDKey struct{}

const TokenCookieName = "token"

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if isSkipCheckAuth(req.URL.Path) {
			next.ServeHTTP(resp, req)
			return
		}

		tokenCookie, err := req.Cookie(TokenCookieName)
		if err != nil {
			if err == http.ErrNoCookie {
				resp.WriteHeader(http.StatusUnauthorized)
				return
			}

			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		userID, err := jwttoken.Parse(tokenCookie.Value)
		if err != nil {
			resp.WriteHeader(http.StatusUnauthorized)
			return
		}

		req = req.WithContext(context.WithValue(req.Context(), UserIDKey{}, userID))

		next.ServeHTTP(resp, req)
	})
}

func isSkipCheckAuth(urlPath string) bool {
	return strings.Contains(urlPath, "api/user/register") || strings.Contains(urlPath, "api/user/login")
}
