package main

import (
	"fmt"
	"net/http"
)

func handlePostToken(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "issue token handler\n")
}
