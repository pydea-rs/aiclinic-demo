package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Session is an authenticated (or anonymous) client against the backend. It
// owns a cookie jar so the session cookie ("sid") and the CSRF cookie
// ("csrf-token") persist across requests, exactly like a browser.
type Session struct {
	baseURL    string
	http       *http.Client
	maxRetries int
}

// envelope is the standard success/error response wrapper used by the backend.
type envelope struct {
	Status   int             `json:"status"`
	Message  json.RawMessage `json:"message"`
	Contents json.RawMessage `json:"contents"`
}

func newSession(cfg *Config) (*Session, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &Session{
		baseURL: cfg.ServerURL,
		http: &http.Client{
			Jar:     jar,
			Timeout: time.Duration(cfg.RequestTimeout) * time.Second,
		},
		maxRetries: cfg.MaxRetries,
	}, nil
}

// prime performs a safe GET so the server issues a csrf-token cookie into the
// jar before any mutation is attempted (mutations require the double-submit
// token, and the very first request never has it yet).
func (s *Session) prime() error {
	return s.do(http.MethodGet, "/", nil, nil)
}

// csrfToken returns the current csrf-token cookie value from the jar, if any.
func (s *Session) csrfToken() string {
	u, err := url.Parse(s.baseURL)
	if err != nil {
		return ""
	}
	for _, c := range s.http.Jar.Cookies(u) {
		if c.Name == "csrf-token" {
			return c.Value
		}
	}
	return ""
}

// do performs an HTTP request, unwraps the response envelope into `out` (when
// non-nil), and retries on 429 / 5xx with exponential backoff. A safe method is
// used to seed the CSRF cookie; other methods send the X-CSRF-Token header.
func (s *Session) do(method, path string, body any, out any) error {
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		payload = b
	}

	var lastErr error
	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * 250 * time.Millisecond
			if backoff > 8*time.Second {
				backoff = 8 * time.Second
			}
			// add a little jitter
			backoff += time.Duration(rng.intn(200)) * time.Millisecond
			time.Sleep(backoff)
		}

		req, err := http.NewRequest(method, s.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
			if tok := s.csrfToken(); tok != "" {
				req.Header.Set("X-CSRF-Token", tok)
			}
		}

		resp, err := s.http.Do(req)
		if err != nil {
			lastErr = err
			continue // network error — retry
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, extractMessage(respBody))
			continue // transient — retry
		}

		if resp.StatusCode >= 400 {
			return &APIError{Status: resp.StatusCode, Message: extractMessage(respBody)}
		}

		if out != nil && len(respBody) > 0 {
			var env envelope
			if err := json.Unmarshal(respBody, &env); err == nil && env.Contents != nil {
				return json.Unmarshal(env.Contents, out)
			}
			// Fallback: body wasn't an envelope — decode directly.
			return json.Unmarshal(respBody, out)
		}
		return nil
	}
	return fmt.Errorf("request %s %s failed after %d attempts: %w", method, path, s.maxRetries+1, lastErr)
}

// APIError carries the backend's HTTP status and human-readable message.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.Status, e.Message) }

// isConflict reports whether an error is a 409 (already-exists) response.
func isConflict(err error) bool {
	var ae *APIError
	if e, ok := err.(*APIError); ok {
		ae = e
	}
	return ae != nil && ae.Status == http.StatusConflict
}

// extractMessage pulls a readable message out of an error envelope, coping with
// both string and string[] (class-validator) message shapes.
func extractMessage(body []byte) string {
	if len(body) == 0 {
		return "(empty response)"
	}
	var m struct {
		Message json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(body, &m); err != nil || m.Message == nil {
		return strings.TrimSpace(string(body))
	}
	var asString string
	if json.Unmarshal(m.Message, &asString) == nil {
		return asString
	}
	var asArray []string
	if json.Unmarshal(m.Message, &asArray) == nil {
		return strings.Join(asArray, "; ")
	}
	return strings.TrimSpace(string(m.Message))
}
