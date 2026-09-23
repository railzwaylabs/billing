package http

import (
	"net/http"

	"github.com/railzwaylabs/billing/internal/shared/transport/httpresponse"
)

func writeError(w http.ResponseWriter, err error) {
	httpresponse.WriteError(w, err)
}
