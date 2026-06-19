// Command server is a minimal Nais example application.
//
// It prints a greeting on "/" and exposes a health endpoint on "/healthz".
// It reports the environment (NAIS_CLUSTER_NAME, injected by Nais) and whether a
// Valkey instance is wired in, using the environment variables Nais injects for
// the "cache" instance (see .nais/valkey.yaml).
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		env := getenv("NAIS_CLUSTER_NAME", "local")

		fmt.Fprintln(w, "Hello Nais")
		fmt.Fprintf(w, "environment: %s\n", env)

		// VALKEY_HOST_CACHE is injected by Nais when the "cache" Valkey instance
		// is referenced from the workload manifest. We only surface it here to
		// prove the wiring; a real app would connect using a Valkey client.
		if host := os.Getenv("VALKEY_HOST_CACHE"); host != "" {
			fmt.Fprintf(w, "valkey: connected to %s\n", host)
		} else {
			fmt.Fprintln(w, "valkey: not configured")
		}
	})

	port := getenv("PORT", "8080")
	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
