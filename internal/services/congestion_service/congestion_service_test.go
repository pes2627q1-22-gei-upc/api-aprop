package congestion_service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"api-aprop/internal/models/congestion_model"
)

type areaCall struct {
	lat, lng, radius float64
	from, to         time.Time
}

type fakeRepository struct {
	def   congestion_model.AreaStats
	err   error
	calls []areaCall
}

func (f *fakeRepository) AreaStats(_ context.Context, lat, lng, radiusKm float64, from, to time.Time) (congestion_model.AreaStats, error) {
	f.calls = append(f.calls, areaCall{lat: lat, lng: lng, radius: radiusKm, from: from, to: to})
	if f.err != nil {
		return congestion_model.AreaStats{}, f.err
	}
	return f.def, nil
}

var (
	madrid  = mustLocation("Europe/Madrid")
	fixedAt = time.Date(2026, 10, 15, 22, 30, 0, 0, time.UTC) // 16/10 00:30 in Madrid
)

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func ptr[T any](v T) *T { return &v }

func newService(repo *fakeRepository) *CongestionService {
	return NewCongestionService(repo, func() time.Time { return fixedAt }, madrid, 0)
}

func location(id string, lat, lng float64) congestion_model.Location {
	return congestion_model.Location{ID: id, Latitude: ptr(lat), Longitude: ptr(lng)}
}

func TestEstimateDefaults(t *testing.T) {
	repo := &fakeRepository{}
	res, err := newService(repo).Estimate(context.Background(), congestion_model.EstimateRequest{
		Locations: []congestion_model.Location{location("punt_1", 41.3871, 2.1701)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.calls) != 1 {
		t.Fatalf("expected 1 repository call, got %d", len(repo.calls))
	}
	call := repo.calls[0]
	if call.radius != DefaultRadiusKm {
		t.Errorf("radius = %v, want default %v", call.radius, DefaultRadiusKm)
	}
	// No timeframe means now, as a day of the service location
	if call.from.Day() != 16 || call.to.Day() != 16 || call.from.Location() != madrid {
		t.Errorf("timeframe = %v..%v, want 16/10 in Europe/Madrid", call.from, call.to)
	}
	want := congestion_model.Result{ID: "punt_1", Latitude: 41.3871, Longitude: 2.1701, CongestionFactor: 0.1, Level: congestion_model.LevelLow}
	if len(res.Results) != 1 || res.Results[0] != want {
		t.Errorf("results = %+v, want [%+v]", res.Results, want)
	}
}

func TestEstimateTimeframe(t *testing.T) {
	start := time.Date(2026, 10, 15, 18, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 17, 20, 0, 0, 0, time.UTC) // 22:00 in Madrid

	tests := []struct {
		name             string
		tf               *congestion_model.Timeframe
		wantFrom, wantTo int // day of month in Madrid
	}{
		{"both", &congestion_model.Timeframe{StartTime: &start, EndTime: &end}, 15, 17},
		{"only start", &congestion_model.Timeframe{StartTime: &end}, 17, 17},
		{"only end", &congestion_model.Timeframe{EndTime: &end}, 16, 17},
		{"empty", &congestion_model.Timeframe{}, 16, 16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{}
			_, err := newService(repo).Estimate(context.Background(), congestion_model.EstimateRequest{
				Timeframe: tt.tf,
				Locations: []congestion_model.Location{location("a", 41, 2)},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := repo.calls[0]; got.from.Day() != tt.wantFrom || got.to.Day() != tt.wantTo {
				t.Errorf("days = %d..%d, want %d..%d", got.from.Day(), got.to.Day(), tt.wantFrom, tt.wantTo)
			}
		})
	}
}

func TestEstimateSingleLocationAndOrder(t *testing.T) {
	repo := &fakeRepository{def: congestion_model.AreaStats{EventsCount: 4, ExpectedAttendance: 30}}
	single := location("single", 10, 10)
	res, err := newService(repo).Estimate(context.Background(), congestion_model.EstimateRequest{
		Location:  &single,
		Locations: []congestion_model.Location{location("b", 20, 20), location("c", 30, 30)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ids := []string{}
	for _, r := range res.Results {
		ids = append(ids, r.ID)
		if r.EventsCount != 4 {
			t.Errorf("%s events_count = %d, want 4", r.ID, r.EventsCount)
		}
		// score 34 with K 20: 0.1 + 0.9·(1 − e^(−1.7)) = 0.8355… → 0.84
		if r.CongestionFactor != 0.84 || r.Level != congestion_model.LevelHigh {
			t.Errorf("%s = %v %s, want 0.84 HIGH", r.ID, r.CongestionFactor, r.Level)
		}
	}
	if len(ids) != 3 || ids[0] != "single" || ids[1] != "b" || ids[2] != "c" {
		t.Errorf("ids = %v, want [single b c]", ids)
	}
}

func TestEstimateValidation(t *testing.T) {
	start := time.Date(2026, 10, 15, 18, 0, 0, 0, time.UTC)
	before := start.Add(-time.Hour)
	many := make([]congestion_model.Location, MaxLocations+1)
	for i := range many {
		many[i] = location("x", 1, 1)
	}

	tests := []struct {
		name string
		req  congestion_model.EstimateRequest
		want error
	}{
		{"no locations", congestion_model.EstimateRequest{}, ErrNoLocations},
		{"too many", congestion_model.EstimateRequest{Locations: many}, ErrTooManyLocations},
		{"end before start", congestion_model.EstimateRequest{
			Timeframe: &congestion_model.Timeframe{StartTime: &start, EndTime: &before},
			Locations: []congestion_model.Location{location("a", 1, 1)},
		}, ErrInvalidTimeframe},
		{"missing id", congestion_model.EstimateRequest{Locations: []congestion_model.Location{location(" ", 1, 1)}}, ErrInvalidLocation},
		{"missing latitude", congestion_model.EstimateRequest{Locations: []congestion_model.Location{{ID: "a", Longitude: ptr(1.0)}}}, ErrInvalidLocation},
		{"latitude out of range", congestion_model.EstimateRequest{Locations: []congestion_model.Location{location("a", 91, 1)}}, ErrInvalidLocation},
		{"longitude out of range", congestion_model.EstimateRequest{Locations: []congestion_model.Location{location("a", 1, -181)}}, ErrInvalidLocation},
		{"negative radius", congestion_model.EstimateRequest{Locations: []congestion_model.Location{{ID: "a", Latitude: ptr(1.0), Longitude: ptr(1.0), RadiusKm: ptr(-1.0)}}}, ErrInvalidLocation},
		{"radius too big", congestion_model.EstimateRequest{Locations: []congestion_model.Location{{ID: "a", Latitude: ptr(1.0), Longitude: ptr(1.0), RadiusKm: ptr(MaxRadiusKm + 1)}}}, ErrInvalidLocation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{}
			_, err := newService(repo).Estimate(context.Background(), tt.req)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if len(repo.calls) != 0 {
				t.Errorf("repository called %d times on invalid input", len(repo.calls))
			}
		})
	}
}

func TestEstimateRepositoryError(t *testing.T) {
	repoErr := errors.New("db down")
	_, err := newService(&fakeRepository{err: repoErr}).Estimate(context.Background(), congestion_model.EstimateRequest{
		Locations: []congestion_model.Location{location("a", 1, 1)},
	})
	if !errors.Is(err, repoErr) {
		t.Fatalf("err = %v, want %v", err, repoErr)
	}
}

func TestCongestionFactor(t *testing.T) {
	tests := []struct {
		score float64
		want  float64
	}{
		{0, 0.1},
		{-5, 0.1},
		{1, 0.14},
		{20, 0.67},
		{1e9, 1.0},
	}
	for _, tt := range tests {
		if got := congestionFactor(tt.score, DefaultK); got != tt.want {
			t.Errorf("congestionFactor(%v) = %v, want %v", tt.score, got, tt.want)
		}
	}
	for s := 0.0; s < 500; s += 0.5 {
		f := congestionFactor(s, DefaultK)
		if f < 0.1 || f > 1.0 || math.IsNaN(f) {
			t.Fatalf("congestionFactor(%v) = %v out of [0.1, 1.0]", s, f)
		}
	}
}

func TestLevelFor(t *testing.T) {
	tests := []struct {
		factor float64
		want   congestion_model.Level
	}{
		{0.1, congestion_model.LevelLow},
		{0.39, congestion_model.LevelLow},
		{0.4, congestion_model.LevelMedium},
		{0.69, congestion_model.LevelMedium},
		{0.7, congestion_model.LevelHigh},
		{1.0, congestion_model.LevelHigh},
	}
	for _, tt := range tests {
		if got := levelFor(tt.factor); got != tt.want {
			t.Errorf("levelFor(%v) = %s, want %s", tt.factor, got, tt.want)
		}
	}
}

func TestNewCongestionServiceInvalidK(t *testing.T) {
	for _, k := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if s := NewCongestionService(&fakeRepository{}, time.Now, nil, k); s.k != DefaultK || s.loc != time.UTC {
			t.Errorf("k=%v: got k=%v loc=%v, want DefaultK and UTC", k, s.k, s.loc)
		}
	}
}
