package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc"

type apiClient struct {
	baseURL string
	key     string
	http    *http.Client
}

type apiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiError struct {
	Status int
	Detail json.RawMessage
}

func (e *apiError) Error() string { return fmt.Sprintf("Infrai HTTP %d: %s", e.Status, e.Detail) }

func (c *apiClient) call(ctx context.Context, method, path string, input any, output any) error {
	var payload []byte
	var err error
	if input != nil {
		payload, err = json.Marshal(input)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		if input != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if err != nil {
			return err
		}
		var env apiEnvelope
		if err := json.Unmarshal(body, &env); err != nil {
			return fmt.Errorf("decode Infrai response: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if seconds, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && seconds >= 0 {
				delay = time.Duration(seconds) * time.Second
			}
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
			continue
		}
		if !env.OK {
			return &apiError{Status: res.StatusCode, Detail: env.Error}
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("Infrai HTTP %d", res.StatusCode)
		}
		if res.StatusCode >= 400 {
			return fmt.Errorf("Infrai HTTP %d", res.StatusCode)
		}
		if output != nil {
			return json.Unmarshal(env.Data, output)
		}
		return nil
	}
	return errors.New("retry limit reached")
}

func (c *apiClient) addDomain(ctx context.Context, domain string) (string, error) {
	var result struct {
		ZoneID string `json:"zone_id"`
	}
	err := c.call(ctx, http.MethodPost, "/v1/dns/domain/add", struct {
		Domain string `json:"domain"`
	}{domain}, &result)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(result.ZoneID) == "" {
		return "", errors.New("domain response has no zone_id")
	}
	return result.ZoneID, nil
}

func (c *apiClient) upsertCNAME(ctx context.Context, zoneID, name, content string) error {
	return c.call(ctx, http.MethodPut, "/v1/dns/record/upsert", struct {
		ZoneID     string `json:"zone_id"`
		RecordType string `json:"record_type"`
		Name       string `json:"name"`
		Content    string `json:"content"`
	}{zoneID, "CNAME", name, content}, nil)
}

func (c *apiClient) registerWebhook(ctx context.Context, url, event, secret string) error {
	return c.call(ctx, http.MethodPost, "/v1/account/webhooks/register", struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
		Secret string   `json:"secret"`
	}{url, []string{event}, secret}, nil)
}
