package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
)

type organization struct {
	ID        string              `json:"id"`
	Domain    string              `json:"domain"`
	ZoneID    string              `json:"zone_id"`
	State     string              `json:"state"`
	Receipts  []donorReceipt      `json:"receipts"`
	Reminders []volunteerReminder `json:"reminders"`
	Reports   []campaignReport    `json:"reports"`
}
type donorReceipt struct {
	DonorID     string `json:"donor_id"`
	AmountCents int64  `json:"amount_cents"`
}
type volunteerReminder struct {
	VolunteerID string `json:"volunteer_id"`
	Shift       string `json:"shift"`
}
type campaignReport struct {
	CampaignID  string `json:"campaign_id"`
	RaisedCents int64  `json:"raised_cents"`
}

type store struct {
	mu   sync.Mutex
	orgs map[string]*organization
}

func (s *store) enroll(id, domain, zone string) (organization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.orgs[id]; existing != nil {
		if existing.Domain != domain {
			return organization{}, errors.New("organization already has another domain")
		}
		return *existing, nil
	}
	o := &organization{ID: id, Domain: domain, ZoneID: zone, State: "pending_verification", Receipts: []donorReceipt{}, Reminders: []volunteerReminder{}, Reports: []campaignReport{}}
	s.orgs[id] = o
	return *o, nil
}

func (s *store) verified(domain string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, o := range s.orgs {
		if o.Domain == domain && o.State == "pending_verification" {
			o.State = "active"
			return true
		}
	}
	return false
}

func (s *store) snapshot(id string) (organization, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o := s.orgs[id]
	if o == nil {
		return organization{}, false
	}
	copy := *o
	return copy, true
}

func validSignature(body []byte, header, secret string) bool {
	provided, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
