package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/diegoafg1009/PuntoVecinal/backend/internal/platform/config"
)

type recorded struct {
	method, path, query, contentType, acl string
	body                                  string
}

func fakeS3(t *testing.T) (*S3, func() []recorded) {
	t.Helper()
	var mu sync.Mutex
	var reqs []recorded
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		reqs = append(reqs, recorded{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Content-Type"), r.Header.Get("X-Amz-Acl"), string(body)})
		mu.Unlock()
		if r.Method == http.MethodPost { // DeleteObjects
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><DeleteResult/>`)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	s := NewS3(config.S3{Endpoint: srv.URL, Region: "us-east-1", Bucket: "bucket", AccessKey: "k", SecretKey: "s", ForcePathStyle: true})
	return s, func() []recorded {
		mu.Lock()
		defer mu.Unlock()
		return append([]recorded(nil), reqs...)
	}
}

func TestPut(t *testing.T) {
	s, got := fakeS3(t)
	if err := s.Put(context.Background(), "reports/1/1.jpg", []byte("jpegdata")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	reqs := got()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	r := reqs[0]
	if r.method != http.MethodPut || r.path != "/bucket/reports/1/1.jpg" {
		t.Errorf("request = %s %s", r.method, r.path)
	}
	if r.contentType != "image/jpeg" || r.acl != "private" {
		t.Errorf("content type %q, acl %q", r.contentType, r.acl)
	}
	if !strings.Contains(r.body, "jpegdata") {
		t.Errorf("body %q does not carry the data", r.body)
	}
}

func TestDelete(t *testing.T) {
	s, got := fakeS3(t)
	if err := s.Delete(context.Background(), "reports/1/1.jpg", "reports/1/2.jpg"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	reqs := got()
	if len(reqs) != 1 || reqs[0].method != http.MethodPost || !strings.Contains(reqs[0].query, "delete") {
		t.Fatalf("requests = %+v, want one multi-object delete", reqs)
	}
	for _, key := range []string{"reports/1/1.jpg", "reports/1/2.jpg"} {
		if !strings.Contains(reqs[0].body, key) {
			t.Errorf("delete body does not list %s: %s", key, reqs[0].body)
		}
	}
}

func TestDeleteNoKeysSendsNothing(t *testing.T) {
	s, got := fakeS3(t)
	if err := s.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := len(got()); n != 0 {
		t.Errorf("requests = %d, want 0", n)
	}
}

func TestPutFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusForbidden)
	}))
	defer srv.Close()
	s := NewS3(config.S3{Endpoint: srv.URL, Region: "us-east-1", Bucket: "b", AccessKey: "k", SecretKey: "s", ForcePathStyle: true})
	if err := s.Put(context.Background(), "k", []byte("x")); err == nil {
		t.Error("expected an error")
	}
}

func TestURLUsesPublicEndpointAndExpiry(t *testing.T) {
	s := NewS3(config.S3{
		Endpoint:       "http://rustfs:9000",
		PublicEndpoint: "http://127.0.0.1:9000",
		Region:         "us-east-1",
		Bucket:         "bucket",
		AccessKey:      "k",
		SecretKey:      "s",
		ForcePathStyle: true,
	})

	raw, err := s.URL(context.Background(), "reports/1/1.jpg", time.Hour)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "127.0.0.1:9000" {
		t.Errorf("host = %q, want the public endpoint", u.Host)
	}
	if u.Path != "/bucket/reports/1/1.jpg" {
		t.Errorf("path = %q, want path-style /bucket/key", u.Path)
	}
	q := u.Query()
	if q.Get("X-Amz-Expires") != "3600" {
		t.Errorf("X-Amz-Expires = %q, want 3600", q.Get("X-Amz-Expires"))
	}
	if q.Get("X-Amz-Signature") == "" {
		t.Error("link is not signed")
	}
}

func TestURLFallsBackToEndpointAndVirtualHost(t *testing.T) {
	s := NewS3(config.S3{
		Endpoint:  "https://s3.example.com",
		Region:    "us-east-1",
		Bucket:    "bucket",
		AccessKey: "k",
		SecretKey: "s",
	})

	raw, err := s.URL(context.Background(), "reports/1/1.jpg", 10*time.Minute)
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	u, _ := url.Parse(raw)
	if u.Host != "bucket.s3.example.com" || u.Path != "/reports/1/1.jpg" {
		t.Errorf("url = %s, want virtual-hosted bucket.s3.example.com/reports/1/1.jpg", raw)
	}
	if got := u.Query().Get("X-Amz-Expires"); got != "600" {
		t.Errorf("X-Amz-Expires = %q, want 600", got)
	}
}
