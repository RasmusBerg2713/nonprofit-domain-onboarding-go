package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	secret := os.Getenv("INFRAI_WEBHOOK_SECRET")
	url := os.Getenv("WEBHOOK_URL")
	event := os.Getenv("INFRAI_VERIFICATION_EVENT")
	target := os.Getenv("PRODUCT_CNAME_TARGET")
	if key == "" || secret == "" || url == "" || event == "" || target == "" {
		log.Fatal("set INFRAI_API_KEY, INFRAI_WEBHOOK_SECRET, WEBHOOK_URL, INFRAI_VERIFICATION_EVENT and PRODUCT_CNAME_TARGET")
	}
	client := &apiClient{baseURL: baseURL, key: key, http: &http.Client{}}
	if err := client.registerWebhook(context.Background(), url, event, secret); err != nil {
		log.Fatal(err)
	}
	s := &store{orgs: make(map[string]*organization)}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /organizations", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ID        string `json:"id"`
			Domain    string `json:"domain"`
			CNAMEName string `json:"cname_name"`
		}
		if json.NewDecoder(r.Body).Decode(&input) != nil || input.ID == "" || input.Domain == "" || input.CNAMEName == "" {
			respond(w, 400, map[string]string{"error": "id, domain and cname_name required"})
			return
		}
		if old, ok := s.snapshot(input.ID); ok {
			if old.Domain != input.Domain {
				respond(w, 409, map[string]string{"error": "organization already has another domain"})
			} else {
				respond(w, 200, old)
			}
			return
		}
		zone, err := client.addDomain(r.Context(), input.Domain)
		if err == nil {
			err = client.upsertCNAME(r.Context(), zone, input.CNAMEName, target)
		}
		if err != nil {
			upstreamError(w, err)
			return
		}
		org, err := s.enroll(input.ID, input.Domain, zone)
		if err != nil {
			respond(w, 409, map[string]string{"error": err.Error()})
			return
		}
		respond(w, 201, org)
	})
	mux.HandleFunc("GET /organizations/{id}", func(w http.ResponseWriter, r *http.Request) {
		org, ok := s.snapshot(r.PathValue("id"))
		if !ok {
			respond(w, 404, map[string]string{"error": "organization not found"})
			return
		}
		respond(w, 200, org)
	})
	mux.HandleFunc("POST /webhooks/verification", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil || !validSignature(body, r.Header.Get("X-Infrai-Signature"), secret) {
			respond(w, 401, map[string]string{"error": "invalid signature"})
			return
		}
		var event struct {
			Domain string `json:"domain"`
			Status string `json:"status"`
		}
		if json.Unmarshal(body, &event) != nil || event.Domain == "" {
			respond(w, 400, map[string]string{"error": "domain required"})
			return
		}
		if event.Status == "verified" {
			s.verified(strings.ToLower(event.Domain))
		}
		respond(w, 200, map[string]string{"result": "accepted"})
	})
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Fatal(http.ListenAndServe(addr, mux))
}

func upstreamError(w http.ResponseWriter, err error) {
	var apiErr *apiError
	if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
		respond(w, apiErr.Status, map[string]any{"error": json.RawMessage(apiErr.Detail)})
		return
	}
	respond(w, http.StatusBadGateway, map[string]string{"error": "upstream request failed"})
}
