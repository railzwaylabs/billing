package httpresponse

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/railzwaylabs/billing/internal/shared/apperror"
)

type ErrorDetail struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

type ErrorBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func WriteError(w http.ResponseWriter, err error) {
	status, response := FromError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func FromError(err error) (int, ErrorResponse) {
	status := http.StatusInternalServerError
	body := ErrorBody{
		Code:    "INTERNAL_ERROR",
		Message: "Internal server error",
		Details: []ErrorDetail{},
	}

	var coded apperror.Coded
	if errors.As(err, &coded) {
		status = statusFromKind(coded.Kind())
		body.Code = coded.Code()
		body.Message = coded.Message()
		body.Details = make([]ErrorDetail, 0, len(coded.Details()))

		for _, detail := range coded.Details() {
			body.Details = append(body.Details, ErrorDetail{
				Field: detail.Field,
				Value: detail.Value,
			})
		}
	}

	return status, ErrorResponse{Error: body}
}

func statusFromKind(kind apperror.Kind) int {
	switch kind {
	case apperror.KindUnauthenticated:
		return http.StatusUnauthorized
	case apperror.KindForbidden:
		return http.StatusForbidden
	case apperror.KindNotFound:
		return http.StatusNotFound
	case apperror.KindInvalid:
		return http.StatusUnprocessableEntity
	case apperror.KindConflict:
		return http.StatusConflict
	case apperror.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
