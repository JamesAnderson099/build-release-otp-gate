package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/infrai-examples/build-release-otp-gate/internal/infrai"
	"github.com/infrai-examples/build-release-otp-gate/internal/release"
)

type codeRequest struct {
	BuildID string `json:"build_id"`
	Code    string `json:"code"`
}

func main() {
	client, err := infrai.NewClient(os.Getenv("INFRAI_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	gate := release.NewGate(client)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /release/code", func(w http.ResponseWriter, r *http.Request) {
		var input codeRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.BuildID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "build_id is required"})
			return
		}
		diagnostic, err := gate.RequestCode(r.Context(), input.BuildID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, diagnostic)
	})
	mux.HandleFunc("POST /release/verify", func(w http.ResponseWriter, r *http.Request) {
		var input codeRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.BuildID == "" || input.Code == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "build_id and code are required"})
			return
		}
		diagnostic, err := gate.VerifyAndRelease(r.Context(), input.BuildID, input.Code)
		if err != nil {
			if errors.Is(err, release.ErrBuildNotFound) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusUnauthorized, diagnostic)
			return
		}
		writeJSON(w, http.StatusOK, diagnostic)
	})

	addr := ":8080"
	log.Printf("release gate listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
