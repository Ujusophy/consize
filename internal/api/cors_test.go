package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLocalFrontendOrigins(t *testing.T) {
	for _, origin := range []string{"http://127.0.0.1:3030", "http://localhost:3030", "http://127.0.0.1:3031", "http://localhost:3031"} {
		for _, method := range []string{http.MethodGet, http.MethodOptions} {
			t.Run(origin+method, func(t *testing.T) {
				s := &Server{}
				called := false
				handler := s.withCORS(func(w http.ResponseWriter, r *http.Request) {
					called = true
					w.WriteHeader(http.StatusOK)
				})
				r := httptest.NewRequest(method, "/api/dashboard", nil)
				r.Header.Set("Origin", origin)
				r.Header.Set("Access-Control-Request-Method", http.MethodPost)
				w := httptest.NewRecorder()
				handler(w, r)
				want := http.StatusOK
				if method == http.MethodOptions {
					want = http.StatusNoContent
				}
				if w.Code != want || w.Header().Get("Access-Control-Allow-Origin") != origin {
					t.Fatalf("status=%d headers=%v", w.Code, w.Header())
				}
				if called != (method != http.MethodOptions) {
					t.Fatal("unexpected handler dispatch")
				}
			})
		}
	}
}

func TestConfiguredOriginsOverrideLocalDefaults(t *testing.T) {
	s, _, _ := apiFixture(t)
	s.cfg.AllowedOrigins = []string{"https://consize.example"}
	for _, origin := range []string{"http://127.0.0.1:3031", "https://untrusted.example", "http://127.0.0.1:3031.evil.example"} {
		r := httptest.NewRequest(http.MethodOptions, "/api/dashboard", nil)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		s.withCORS(func(http.ResponseWriter, *http.Request) { t.Fatal("rejected origin dispatched") })(w, r)
		if w.Code != http.StatusForbidden || w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("origin %s was not rejected", origin)
		}
	}
}
