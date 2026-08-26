package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/11DingKing/lab-scheduling/internal/auth"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"github.com/11DingKing/lab-scheduling/internal/service"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	Auth    auth.Service
	Service service.Service
	Queries service.QueryStore
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if s.Queries == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	mux.HandleFunc("/api/v1/auth/login", s.login)
	mux.Handle("/api/v1/equipment", s.Auth.Middleware(http.HandlerFunc(s.equipment)))
	mux.Handle("/api/v1/batches", s.Auth.Middleware(http.HandlerFunc(s.batch)))
	mux.Handle("/api/v1/bookings", s.Auth.Middleware(http.HandlerFunc(s.booking)))
	mux.Handle("/api/v1/reservations", s.Auth.Middleware(http.HandlerFunc(s.reservation)))
	mux.Handle("/api/v1/maintenance", s.Auth.Middleware(auth.Require(http.HandlerFunc(s.maintenance), domain.RoleAdmin, domain.RoleSafety)))
	return requestMiddleware(mux)
}
func requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if strings.TrimSpace(id) == "" {
			id = fmt.Sprintf("req-%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(service.WithRequestID(r.Context(), id)))
	})
}

type loginInput struct{ Email, Password string }

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in loginInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, "invalid_json", err, r.Header.Get("X-Request-ID"))
		return
	}
	token, user, err := s.Auth.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		writeError(w, 401, "unauthorized", err, r.Header.Get("X-Request-ID"))
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "lab_session", Value: token, HttpOnly: true, SameSite: http.SameSiteStrictMode, Path: "/", MaxAge: int(s.Auth.TTL.Seconds())})
	writeJSON(w, 200, map[string]any{"user": user, "expires_in": int(s.Auth.TTL.Seconds())})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func (s *Server) equipment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	items, err := service.ListReadyEquipment(r.Context(), s.Queries)
	if err != nil {
		writeError(w, 500, "query_failed", err, r.Header.Get("X-Request-ID"))
		return
	}
	writeJSON(w, 200, items)
}
func (s *Server) batch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	var in struct {
		ID, Title, Qualification, StartsAt, EndsAt string
		StudentCount                               int
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, "invalid_json", err, r.Header.Get("X-Request-ID"))
		return
	}
	start, _ := time.Parse(time.RFC3339, in.StartsAt)
	end, _ := time.Parse(time.RFC3339, in.EndsAt)
	err := s.Service.CreateBatch(r.Context(), user, domain.CourseBatch{ID: in.ID, Title: in.Title, Qualification: in.Qualification, StudentCount: in.StudentCount, Window: domain.TimeWindow{Start: start, End: end}})
	if err != nil {
		st, code := mapError(err)
		writeError(w, st, code, err, r.Header.Get("X-Request-ID"))
		return
	}
	writeJSON(w, 201, map[string]string{"id": in.ID})
}
func (s *Server) booking(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	var in struct {
		ID, BatchID, EquipmentID, StartsAt, EndsAt, ConsumableID string
		Quantity                                                 int
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, "invalid_json", err, r.Header.Get("X-Request-ID"))
		return
	}
	start, _ := time.Parse(time.RFC3339, in.StartsAt)
	end, _ := time.Parse(time.RFC3339, in.EndsAt)
	_, err := s.Service.SubmitBooking(r.Context(), user, domain.BookingRequest{ID: in.ID, BatchID: in.BatchID, EquipmentID: in.EquipmentID, Window: domain.TimeWindow{Start: start, End: end}}, in.ConsumableID, in.Quantity)
	if err != nil {
		st, code := mapError(err)
		writeError(w, st, code, err, r.Header.Get("X-Request-ID"))
		return
	}
	writeJSON(w, 202, map[string]string{"id": in.ID, "status": "pending"})
}
func (s *Server) reservation(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/reservations/")
	if id == "" {
		w.WriteHeader(404)
		return
	}
	var err error
	if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/checkout") {
		id = strings.TrimSuffix(id, "/checkout")
		var in struct {
			ConsumableID string
			Quantity     int
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		err = s.Service.Checkout(r.Context(), user, id, in.ConsumableID, in.Quantity)
	} else if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/return") {
		id = strings.TrimSuffix(id, "/return")
		err = s.Service.Return(r.Context(), user, id)
	} else {
		rsv, e := s.Service.Store.Reservation(r.Context(), id)
		if e != nil {
			err = e
		} else {
			writeJSON(w, 200, rsv)
			return
		}
	}
	if err != nil {
		st, code := mapError(err)
		writeError(w, st, code, err, r.Header.Get("X-Request-ID"))
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (s *Server) maintenance(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	var in struct{ ID, EquipmentID, Reason, StartsAt, EndsAt string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, "invalid_json", err, r.Header.Get("X-Request-ID"))
		return
	}
	start, _ := time.Parse(time.RFC3339, in.StartsAt)
	end, _ := time.Parse(time.RFC3339, in.EndsAt)
	err := s.Service.AddMaintenance(r.Context(), user, domain.MaintenanceBlock{ID: in.ID, EquipmentID: in.EquipmentID, Reason: in.Reason, Window: domain.TimeWindow{Start: start, End: end}})
	if err != nil {
		st, code := mapError(err)
		writeError(w, st, code, err, r.Header.Get("X-Request-ID"))
		return
	}
	writeJSON(w, 201, map[string]string{"id": in.ID})
}

var _ context.Context
