package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"ulak/internal/config"

	log "github.com/sirupsen/logrus"
)

func ApiKeyAuthMiddleware(cfg config.Auth, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := strings.TrimSpace(r.Header.Get(cfg.ApiHeaderKey))

		if apiKey == "" {
			log.WithFields(map[string]interface{}{
				"method": r.Method,
				"url":    r.URL.String(),
				"ip":     r.RemoteAddr,
			}).Error("Missing ins auth key")
			http.Error(w, "Api key is missing", http.StatusUnauthorized)
			return
		}

		// use ConstantTimeCompare to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(cfg.ApiKey)) != 1 {
			log.WithFields(map[string]interface{}{
				"method": r.Method,
				"url":    r.URL.String(),
				"ip":     r.RemoteAddr,
			})
			http.Error(w, "Api key is invalid", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
