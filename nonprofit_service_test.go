package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerificationDecision(t *testing.T) {
	for _, tc := range []struct {
		name, domain, signature string
		want                    bool
	}{
		{"matching signature activates domain", "help.example.org", "valid", true},
		{"unknown domain stays inactive", "other.example.org", "valid", false},
		{"unsigned event is rejected", "help.example.org", "invalid", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &store{orgs: map[string]*organization{}}
			if _, err := s.enroll("food-bank", "help.example.org", "zone-42"); err != nil {
				t.Fatal(err)
			}
			body := []byte(`{"domain":"` + tc.domain + `","status":"verified"}`)
			mac := hmac.New(sha256.New, []byte("test-secret"))
			mac.Write(body)
			sig := hex.EncodeToString(mac.Sum(nil))
			if tc.signature == "invalid" {
				sig = "invalid"
			}
			got := validSignature(body, sig, "test-secret") && s.verified(tc.domain)
			if got != tc.want {
				t.Fatalf("activation = %v, want %v", got, tc.want)
			}
			org, _ := s.snapshot("food-bank")
			if (org.State == "active") != tc.want {
				t.Fatalf("state = %q", org.State)
			}
		})
	}
}
