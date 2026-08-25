package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/11DingKing/lab-scheduling/internal/domain"
	"net/http"
)

type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func writeError(w http.ResponseWriter, status int, code string, err error, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorBody{Code: code, Message: err.Error(), RequestID: requestID})
}
func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrCapacity):
		return http.StatusConflict, "conflict"
	case errors.Is(err, domain.ErrInvalidState):
		return http.StatusUnprocessableEntity, "invalid_state"
	default:
		return http.StatusBadRequest, "invalid_request"
	}
}
