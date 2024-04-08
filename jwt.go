package main

import "net/http"

// jwt verify, extract claims (partic pipeline slug)
func jwtVerificationMiddleware(_ AuthorizationConfig) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Authorization header must be present

			// verify JWT, fail if invalid

			// additional checks:
			// - organization must be valid
			// - audience must be as expected

			// add claims to request context: pipeline, organization

			next.ServeHTTP(w, r)
		})
	}
}
