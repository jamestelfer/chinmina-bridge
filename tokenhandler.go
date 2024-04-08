package main

import (
	"fmt"
	"net/http"
)

func handlePostToken() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "issue token handler\n")
	})
}
