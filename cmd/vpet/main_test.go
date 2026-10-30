package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// The bundle is always called main.js, so a proxy that caches by URL would serve the previous
// release for as long as its own default TTL. Every static response must ask to revalidate.
func TestStaticResponsesRevalidate(t *testing.T) {
	files := fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html>")},
		"main.js":    {Data: []byte("console.log(1)")},
	}
	handler := spaHandler(http.FS(files))

	for _, path := range []string{"/", "/main.js", "/unknown-route"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if got := w.Header().Get("Cache-Control"); got != "no-cache" {
			t.Fatalf("%s: Cache-Control %q, want no-cache", path, got)
		}
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d", path, w.Code)
		}
	}
}

// Revalidation only stays cheap if an unchanged file still answers 304.
func TestUnchangedBundleStillAnswers304(t *testing.T) {
	files := fstest.MapFS{"main.js": {Data: []byte("console.log(1)")}}
	handler := spaHandler(http.FS(files))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/main.js", nil))
	tag := first.Header().Get("ETag")
	if tag == "" {
		t.Fatal("no ETag to revalidate against")
	}

	again := httptest.NewRequest(http.MethodGet, "/main.js", nil)
	again.Header.Set("If-None-Match", tag)
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, again)
	if second.Code != http.StatusNotModified {
		t.Fatalf("status %d, want 304", second.Code)
	}
}

// An unknown path is the single-page app falling back to index.html, not a 404.
func TestUnknownPathServesTheApp(t *testing.T) {
	files := fstest.MapFS{"index.html": {Data: []byte("<!doctype html>app")}}
	w := httptest.NewRecorder()
	spaHandler(http.FS(files)).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/duel/123", nil))
	if w.Code != http.StatusOK || w.Body.String() != "<!doctype html>app" {
		t.Fatalf("status %d body %q", w.Code, w.Body.String())
	}
}
