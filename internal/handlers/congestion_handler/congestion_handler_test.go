package congestion_handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"api-aprop/internal/models/congestion_model"
	"api-aprop/internal/services/congestion_service"
)

type fakeService struct {
	res   congestion_model.EstimateResponse
	err   error
	calls int
	got   congestion_model.EstimateRequest
}

func (f *fakeService) Estimate(_ context.Context, req congestion_model.EstimateRequest) (congestion_model.EstimateResponse, error) {
	f.calls++
	f.got = req
	return f.res, f.err
}

func do(t *testing.T, svc congestion_service.Service, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(EstimatePattern, NewCongestionHandler(svc).Estimate)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/congestion/estimate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// The example of the contract agreed with Mobilicat.
const contractRequest = `{
  "timeframe": {"start_time": "2026-10-15T18:00:00Z", "end_time": "2026-10-15T22:00:00Z"},
  "locations": [{"id": "punt_1", "latitude": 41.3871, "longitude": 2.1701, "radius_km": 1.5}]
}`

func TestEstimateContract(t *testing.T) {
	svc := &fakeService{res: congestion_model.EstimateResponse{Results: []congestion_model.Result{{
		ID: "punt_1", Latitude: 41.3871, Longitude: 2.1701, CongestionFactor: 0.85, Level: congestion_model.LevelHigh, EventsCount: 4,
	}}}}

	rec := do(t, svc, contractRequest)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body)
	}
	// Request decoded with the snake_case keys of the contract
	got := svc.got
	if got.Timeframe == nil || got.Timeframe.StartTime == nil || got.Timeframe.EndTime == nil {
		t.Fatalf("timeframe not decoded: %+v", got.Timeframe)
	}
	if !got.Timeframe.StartTime.Equal(time.Date(2026, 10, 15, 18, 0, 0, 0, time.UTC)) {
		t.Errorf("start_time = %v", got.Timeframe.StartTime)
	}
	if len(got.Locations) != 1 || got.Locations[0].ID != "punt_1" || *got.Locations[0].RadiusKm != 1.5 ||
		*got.Locations[0].Latitude != 41.3871 || *got.Locations[0].Longitude != 2.1701 {
		t.Errorf("locations not decoded: %+v", got.Locations)
	}

	// Response encoded with the snake_case keys of the contract
	var body map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := map[string]any{
		"id": "punt_1", "latitude": 41.3871, "longitude": 2.1701,
		"congestion_factor": 0.85, "level": "HIGH", "events_count": float64(4),
	}
	if len(body["results"]) != 1 {
		t.Fatalf("results = %v", body["results"])
	}
	for k, v := range want {
		if body["results"][0][k] != v {
			t.Errorf("results[0][%q] = %v, want %v", k, body["results"][0][k], v)
		}
	}
}

func TestEstimateSingleLocation(t *testing.T) {
	svc := &fakeService{}
	rec := do(t, svc, `{"location": {"id": "a", "latitude": 41, "longitude": 2}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if svc.got.Location == nil || svc.got.Location.ID != "a" || svc.got.Location.RadiusKm != nil {
		t.Errorf("location = %+v", svc.got.Location)
	}
	if svc.got.Timeframe != nil {
		t.Errorf("timeframe = %+v, want nil", svc.got.Timeframe)
	}
}

func TestEstimateInvalidBody(t *testing.T) {
	for _, body := range []string{``, `not json`, `{"locations": "x"}`, `{"timeframe": {"start_time": "yesterday"}}`} {
		svc := &fakeService{}
		rec := do(t, svc, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", body, rec.Code)
		}
		if svc.calls != 0 {
			t.Errorf("body %q: service called", body)
		}
	}
}

func TestEstimateServiceErrors(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{congestion_service.ErrNoLocations, http.StatusBadRequest},
		{fmt.Errorf("%w: at most 100", congestion_service.ErrTooManyLocations), http.StatusBadRequest},
		{fmt.Errorf("%w: locations[0]: id is required", congestion_service.ErrInvalidLocation), http.StatusBadRequest},
		{congestion_service.ErrInvalidTimeframe, http.StatusBadRequest},
		{errors.New("db down"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		rec := do(t, &fakeService{err: tt.err}, contractRequest)
		if rec.Code != tt.want {
			t.Errorf("%v: status = %d, want %d", tt.err, rec.Code, tt.want)
		}
		if tt.want == http.StatusInternalServerError && strings.Contains(rec.Body.String(), "db down") {
			t.Errorf("internal error leaked: %s", rec.Body)
		}
	}
}
