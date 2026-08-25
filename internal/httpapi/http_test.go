package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/11DingKing/lab-scheduling/internal/auth"
	"github.com/11DingKing/lab-scheduling/internal/clock"
	"github.com/11DingKing/lab-scheduling/internal/httpapi"
	"github.com/11DingKing/lab-scheduling/internal/service"
	"github.com/11DingKing/lab-scheduling/internal/testutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func server(t *testing.T) *httpapi.Server {
	db := testutil.Open(t)
	t.Helper()
	return &httpapi.Server{Auth: auth.Service{Store: db, Clock: clock.Real{}, TTL: time.Hour}, Service: service.Service{Store: db, Clock: clock.Real{}}, Queries: db}
}
func login(t *testing.T, h http.Handler, email, password string) (string, int) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	r := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var out struct {
		User struct {
			Role string `json:"role"`
		} `json:"user"`
	}
	_ = json.NewDecoder(w.Body).Decode(&out)
	return w.Header().Get("Set-Cookie"), w.Code
}
func TestHealthAndReady(t *testing.T) {
	s := server(t)
	h := s.Handler()
	for _, path := range []string{"/healthz", "/readyz"} {
		r := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s=%d", path, w.Code)
		}
	}
}
func TestLoginSuccessAndFailure(t *testing.T) {
	s := server(t)
	h := s.Handler()
	cookie, status := login(t, h, "admin@example.com", "admin")
	if status != 200 || cookie == "" {
		t.Fatalf("status=%d cookie=%q", status, cookie)
	}
	_, status = login(t, h, "admin@example.com", "wrong")
	if status != 401 {
		t.Fatalf("status=%d", status)
	}
}
func TestRequestIDAndUnauthorized(t *testing.T) {
	s := server(t)
	r := httptest.NewRequest("GET", "/api/v1/equipment", nil)
	r.Header.Set("X-Request-ID", "custom")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w.Header().Get("X-Request-ID") != "custom" {
		t.Fatal("request id missing")
	}
}
func TestEquipmentRequiresSession(t *testing.T) {
	s := server(t)
	h := s.Handler()
	cookie, _ := login(t, h, "admin@example.com", "admin")
	r := httptest.NewRequest("GET", "/api/v1/equipment", nil)
	r.Header.Set("Cookie", cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var items []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("empty equipment")
	}
}
func TestBatchEndpointValidation(t *testing.T) {
	s := server(t)
	h := s.Handler()
	cookie, _ := login(t, h, "teacher@example.com", "teacher")
	payload := map[string]any{"id": "http-batch", "title": "HTTP class", "qualification": "CNC-1", "studentCount": 5, "startsAt": time.Now().UTC().Add(time.Hour).Format(time.RFC3339), "endsAt": time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)}
	body, _ := json.Marshal(payload)
	r := httptest.NewRequest("POST", "/api/v1/batches", bytes.NewReader(body))
	r.Header.Set("Cookie", cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	bad := httptest.NewRequest("POST", "/api/v1/batches", bytes.NewReader([]byte("{")))
	bad.Header.Set("Cookie", cookie)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, bad)
	if w.Code != 400 {
		t.Fatalf("bad=%d", w.Code)
	}
}
func TestRoleProtectedMaintenance(t *testing.T) {
	s := server(t)
	h := s.Handler()
	cookie, _ := login(t, h, "teacher@example.com", "teacher")
	r := httptest.NewRequest("POST", "/api/v1/maintenance", bytes.NewReader([]byte(`{"id":"m","equipmentId":"eq-cnc"}`)))
	r.Header.Set("Cookie", cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("status=%d", w.Code)
	}
}
func TestRecoveryMiddleware(t *testing.T) {
	handler := httpapi.Recovery(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 500 {
		t.Fatalf("status=%d", w.Code)
	}
}
func TestMethodHelper(t *testing.T) {
	handler := httpapi.Method("POST", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(201) }))
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 405 || w.Header().Get("Allow") != "POST" {
		t.Fatal(w.Code)
	}
	r = httptest.NewRequest("POST", "/", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatal(w.Code)
	}
}

var _ = context.Background
