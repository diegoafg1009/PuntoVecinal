package embedding

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func vectorOf(n int, v float32) []float32 {
	out := make([]float32, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func TestEmbedSendsOneInputAndParsesTheVector(t *testing.T) {
	var got request
	var auth, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"embedding": vectorOf(Dimensions, 0.5), "index": 0}},
		})
	}))
	defer srv.Close()

	vec, err := NewVoyage("secret", WithBaseURL(srv.URL)).Embed(context.Background(), "Poste sin luz")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}

	if path != "/v1/embeddings" || auth != "Bearer secret" {
		t.Errorf("path %q, authorization %q", path, auth)
	}
	if got.Model != Model || got.InputType != "document" {
		t.Errorf("model %q, input_type %q", got.Model, got.InputType)
	}
	if len(got.Input) != 1 || got.Input[0] != "Poste sin luz" {
		t.Errorf("input = %q, want exactly the text", got.Input)
	}
	if n := len(vec.Slice()); n != Dimensions {
		t.Errorf("vector has %d dimensions, want %d", n, Dimensions)
	}
	if vec.Slice()[0] != 0.5 {
		t.Errorf("first component = %v, want 0.5", vec.Slice()[0])
	}
}

func TestEmbedFailuresAreUnavailable(t *testing.T) {
	tests := map[string]http.HandlerFunc{
		"server error": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		},
		"rate limited": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "slow down", http.StatusTooManyRequests)
		},
		"not json": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html>"))
		},
		"no data": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"data":[]}`))
		},
		"wrong size": func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"embedding": vectorOf(8, 1)}},
			})
		},
	}
	for name, handler := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(handler)
			defer srv.Close()

			_, err := NewVoyage("secret", WithBaseURL(srv.URL)).Embed(context.Background(), "x")
			if !errors.Is(err, ErrUnavailable) {
				t.Fatalf("err = %v, want ErrUnavailable", err)
			}
		})
	}
}

func TestEmbedUnreachableServerIsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens there anymore

	_, err := NewVoyage("secret", WithBaseURL(url)).Embed(context.Background(), "x")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

func TestEmbedWithoutKeyMakesNoRequest(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer srv.Close()

	v := NewVoyage("", WithBaseURL(srv.URL))
	if v.Configured() {
		t.Error("Configured() = true without a key")
	}
	if _, err := v.Embed(context.Background(), "x"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
	if n := calls.Load(); n != 0 {
		t.Errorf("%d requests were made without a key", n)
	}
}

func TestEmbedHonorsContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewVoyage("secret", WithBaseURL(srv.URL)).Embed(ctx, "x"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}
