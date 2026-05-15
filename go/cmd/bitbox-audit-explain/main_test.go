package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildPromptValidJSON(t *testing.T) {
	input := []byte(`{"findings": []}`)
	got, err := buildPrompt(input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "hardware-wallet integration reviewer") {
		t.Errorf("prompt missing template preamble")
	}
	if !strings.Contains(got, `"findings": []`) {
		t.Errorf("prompt missing input JSON")
	}
}

func TestBuildPromptPrettyPrintsInput(t *testing.T) {
	input := []byte(`{"a":1,"b":2}`)
	got, err := buildPrompt(input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "\"a\": 1") {
		t.Errorf("prompt did not pretty-print JSON: %s", got)
	}
}

func TestBuildPromptRejectsInvalidJSON(t *testing.T) {
	_, err := buildPrompt([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestCallClaudeRoundtripSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Errorf("missing/wrong api key header: %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got != anthropicVersion {
			t.Errorf("wrong anthropic-version header: %q", got)
		}
		var body anthropicReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "test-model" {
			t.Errorf("wrong model: %q", body.Model)
		}
		if len(body.Messages) != 1 || body.Messages[0].Content != "hi prompt" {
			t.Errorf("unexpected messages: %+v", body.Messages)
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"narrative reply"}]}`))
	}))
	defer srv.Close()

	got, err := callClaude(srv.Client(), srv.URL, "test-key", "test-model", "hi prompt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "narrative reply" {
		t.Errorf("got %q, want %q", got, "narrative reply")
	}
}

func TestCallClaudeSurfacesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"type":"server_error","message":"boom"}}`))
	}))
	defer srv.Close()

	_, err := callClaude(srv.Client(), srv.URL, "k", "m", "p")
	if err == nil {
		t.Fatal("expected error on 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status: %v", err)
	}
}

func TestCallClaudeSurfacesAPIErrorBody(t *testing.T) {
	// 200 with an `error` body — Anthropic sometimes returns 200 with a
	// structured error (rate limit pre-call). We surface the type/message.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":{"type":"overloaded_error","message":"backoff please"}}`))
	}))
	defer srv.Close()

	_, err := callClaude(srv.Client(), srv.URL, "k", "m", "p")
	if err == nil {
		t.Fatal("expected error on api-error body")
	}
	if !strings.Contains(err.Error(), "overloaded_error") || !strings.Contains(err.Error(), "backoff please") {
		t.Errorf("error message lost details: %v", err)
	}
}

func TestCallClaudeJoinsMultipleContentBlocks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"content":[
			{"type":"text","text":"first "},
			{"type":"text","text":"second"},
			{"type":"thinking","text":"ignored"}
		]}`))
	}))
	defer srv.Close()

	got, err := callClaude(srv.Client(), srv.URL, "k", "m", "p")
	if err != nil {
		t.Fatal(err)
	}
	if got != "first second" {
		t.Errorf("got %q, want concatenated text blocks", got)
	}
}
