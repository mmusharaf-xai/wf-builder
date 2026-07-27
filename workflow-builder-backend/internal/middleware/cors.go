package middleware

import (
	"net/http"
	"strings"
)

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}
	wildcard := containsWildcard(allowedOrigins)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (wildcard || isAllowedOrigin(origin, allowed)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			}

			if r.Method == http.MethodOptions {
				// Always answer preflight so browsers get a clean 204.
				if origin != "" && w.Header().Get("Access-Control-Allow-Origin") == "" {
					// Reflect origin only when explicitly allowed; otherwise omit.
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isAllowedOrigin(origin string, allowed map[string]struct{}) bool {
	if _, ok := allowed[origin]; ok {
		return true
	}
	// Dev convenience: any localhost / 127.0.0.1 port when listed as
	// "http://localhost:*" or when exact match failed but map has a localhost entry.
	// Prefer exact matches from CORS_ORIGINS; also accept common local hosts
	// if any localhost origin is configured.
	if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
		for o := range allowed {
			if strings.HasPrefix(o, "http://localhost") || strings.HasPrefix(o, "http://127.0.0.1") {
				return true
			}
		}
	}
	return false
}

func containsWildcard(origins []string) bool {
	for _, o := range origins {
		if strings.TrimSpace(o) == "*" {
			return true
		}
	}
	return false
}
