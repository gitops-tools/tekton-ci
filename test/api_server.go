package test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MakeAPIServer is used during testing to create an HTTP server to return
// fixtures if the request matches.
func MakeAPIServer(t *testing.T, urlPath, ref, fixture string) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != urlPath {
			t.Fatalf("request path got %s, want %s", r.URL.Path, urlPath)
		}
		if queryRef := r.URL.Query().Get("ref"); queryRef != ref {
			t.Fatalf("failed to match ref, got %s, want %s", queryRef, ref)
		}
		if fixture == "" {
			http.NotFound(w, r)
			return
		}
		b, err := ioutil.ReadFile(fixture)
		if err != nil {
			t.Fatalf("failed to read %s: %s", fixture, err)
		}
		_, err = w.Write(b)
		if err != nil {
			t.Fatalf("failed to write out the body: %s", err)
		}
	}))
}
