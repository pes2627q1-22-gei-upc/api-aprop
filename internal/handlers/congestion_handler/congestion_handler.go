package congestion_handler

import (
	"encoding/json"
	"net/http"

	"api-aprop/internal/handlers/response"
	"api-aprop/internal/models/congestion_model"
	"api-aprop/internal/services/congestion_service"
)

// EstimatePattern is the route served to Mobilicat. It is registered outside the router so it
// is protected by the API key middleware instead of the Firebase one (see cmd/app/main.go).
const EstimatePattern = "POST /api/v1/congestion/estimate"

// maxBodyBytes limits the request body (100 locations fit in a few KB).
const maxBodyBytes = 1 << 20

type CongestionHandler struct {
	svc congestion_service.Service
}

func NewCongestionHandler(svc congestion_service.Service) *CongestionHandler {
	return &CongestionHandler{svc: svc}
}

// Estimate handles POST /api/v1/congestion/estimate.
func (h *CongestionHandler) Estimate(w http.ResponseWriter, r *http.Request) {
	var req congestion_model.EstimateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.svc.Estimate(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, res)
}
