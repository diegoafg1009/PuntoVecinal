// Package embedding turns text into vectors that capture its meaning, using the
// Voyage AI API. Callers decide what text to send; this package never adds any.
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pgvector/pgvector-go"
)

const (
	// DefaultBaseURL is the Voyage AI API.
	DefaultBaseURL = "https://api.voyageai.com"

	// Model is multilingual and returns 1024 dimensions by default. The first
	// 200 million tokens are free for every account.
	Model = "voyage-4"

	// Dimensions is the size of every vector, as the `reports.embedding` column
	// expects.
	Dimensions = 1024

	// inputType is "document" for both stored reports and drafts: detecting
	// duplicates is symmetric, since a draft and a report are the same kind of
	// text.
	inputType = "document"

	// maxResponseBytes bounds what is read from the API: one 1024-dim answer is
	// well under 100 KiB.
	maxResponseBytes = 1 << 20
)

// ErrUnavailable is returned when the embedding cannot be obtained: no API key,
// a transport error, a non-2xx answer or an answer that is not a vector of the
// expected size.
var ErrUnavailable = errors.New("embedding service unavailable")

// Voyage is a client of the Voyage AI embeddings API.
type Voyage struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// Option changes how a Voyage client is built.
type Option func(*Voyage)

// WithBaseURL points the client to another server, for tests.
func WithBaseURL(url string) Option {
	return func(v *Voyage) { v.baseURL = url }
}

// WithHTTPClient replaces the HTTP client, for tests.
func WithHTTPClient(c *http.Client) Option {
	return func(v *Voyage) { v.client = c }
}

// NewVoyage builds a client. With an empty apiKey every call returns
// ErrUnavailable without making a request, so the feature switches itself off
// when no key is configured.
func NewVoyage(apiKey string, opts ...Option) *Voyage {
	v := &Voyage{
		apiKey:  apiKey,
		baseURL: DefaultBaseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Configured tells whether an API key is set.
func (v *Voyage) Configured() bool { return v.apiKey != "" }

type request struct {
	Input     []string `json:"input"`
	Model     string   `json:"model"`
	InputType string   `json:"input_type"`
}

type response struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Embed returns the vector of text. Every failure is ErrUnavailable, wrapped
// with the cause; the text is never included in an error.
func (v *Voyage) Embed(ctx context.Context, text string) (pgvector.Vector, error) {
	if v.apiKey == "" {
		return pgvector.Vector{}, fmt.Errorf("%w: no API key", ErrUnavailable)
	}

	body, err := json.Marshal(request{Input: []string{text}, Model: Model, InputType: inputType})
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("%w: encode request: %v", ErrUnavailable, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("%w: build request: %v", ErrUnavailable, err)
	}
	req.Header.Set("Authorization", "Bearer "+v.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := v.client.Do(req)
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return pgvector.Vector{}, fmt.Errorf("%w: status %d", ErrUnavailable, res.StatusCode)
	}

	var out response
	if err := json.NewDecoder(io.LimitReader(res.Body, maxResponseBytes)).Decode(&out); err != nil {
		return pgvector.Vector{}, fmt.Errorf("%w: decode answer: %v", ErrUnavailable, err)
	}
	if len(out.Data) != 1 || len(out.Data[0].Embedding) != Dimensions {
		return pgvector.Vector{}, fmt.Errorf("%w: unexpected answer shape", ErrUnavailable)
	}
	return pgvector.NewVector(out.Data[0].Embedding), nil
}
