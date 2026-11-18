package congestion_service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"api-aprop/internal/models/congestion_model"
	"api-aprop/internal/repositories/congestion_repository"
)

// TODO(DP-032): defaults, limits and formula proposed by FiveGuys, pending confirmation by Mobilicat.
const (
	// DefaultRadiusKm is used when a location has no radius.
	DefaultRadiusKm = 1.0
	// MaxRadiusKm is the largest radius accepted for a location.
	MaxRadiusKm = 50.0
	// MaxLocations is the largest number of locations in one request.
	MaxLocations = 100
	// DefaultK is the score that gives about 63 % of the congestion range (see congestionFactor).
	DefaultK = 20.0

	minFactor = 0.1
	maxFactor = 1.0

	// Factors from mediumThreshold are MEDIUM and from highThreshold are HIGH.
	mediumThreshold = 0.4
	highThreshold   = 0.7
)

type CongestionService struct {
	repo congestion_repository.Repository
	now  func() time.Time
	loc  *time.Location
	k    float64
}

var _ Service = (*CongestionService)(nil)

// NewCongestionService creates the service. Event dates have no time, so the timeframe is turned
// into calendar days of loc. A k <= 0 falls back to DefaultK.
func NewCongestionService(repo congestion_repository.Repository, now func() time.Time, loc *time.Location, k float64) *CongestionService {
	if loc == nil {
		loc = time.UTC
	}
	if k <= 0 || math.IsNaN(k) || math.IsInf(k, 0) {
		k = DefaultK
	}
	return &CongestionService{repo: repo, now: now, loc: loc, k: k}
}

func (s *CongestionService) Estimate(ctx context.Context, req congestion_model.EstimateRequest) (congestion_model.EstimateResponse, error) {
	from, to, err := s.timeframe(req.Timeframe)
	if err != nil {
		return congestion_model.EstimateResponse{}, err
	}

	locations := req.Locations
	if req.Location != nil {
		locations = append([]congestion_model.Location{*req.Location}, locations...)
	}
	if len(locations) == 0 {
		return congestion_model.EstimateResponse{}, ErrNoLocations
	}
	if len(locations) > MaxLocations {
		return congestion_model.EstimateResponse{}, fmt.Errorf("%w: at most %d", ErrTooManyLocations, MaxLocations)
	}

	// Validate every location before querying the database
	type point struct {
		id            string
		lat, lng, rad float64
	}
	points := make([]point, 0, len(locations))
	for i, l := range locations {
		p, err := validateLocation(l)
		if err != nil {
			return congestion_model.EstimateResponse{}, fmt.Errorf("%w: locations[%d]: %v", ErrInvalidLocation, i, err)
		}
		points = append(points, point{id: p.ID, lat: *p.Latitude, lng: *p.Longitude, rad: *p.RadiusKm})
	}

	results := make([]congestion_model.Result, 0, len(points))
	for _, p := range points {
		stats, err := s.repo.AreaStats(ctx, p.lat, p.lng, p.rad, from, to)
		if err != nil {
			return congestion_model.EstimateResponse{}, fmt.Errorf("estimate congestion of %q: %w", p.id, err)
		}
		factor := congestionFactor(score(stats), s.k)
		results = append(results, congestion_model.Result{
			ID:               p.id,
			Latitude:         p.lat,
			Longitude:        p.lng,
			CongestionFactor: factor,
			Level:            levelFor(factor),
			EventsCount:      stats.EventsCount,
		})
	}
	return congestion_model.EstimateResponse{Results: results}, nil
}

// timeframe applies the defaults: no start means now, no end means the start.
// It returns both ends in the service location.
func (s *CongestionService) timeframe(tf *congestion_model.Timeframe) (time.Time, time.Time, error) {
	now := s.now()
	start, end := now, now
	if tf != nil {
		if tf.StartTime != nil {
			start = *tf.StartTime
			end = start
		}
		if tf.EndTime != nil {
			end = *tf.EndTime
		}
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: end_time is before start_time", ErrInvalidTimeframe)
	}
	return start.In(s.loc), end.In(s.loc), nil
}

// validateLocation checks a location and fills the default radius.
func validateLocation(l congestion_model.Location) (congestion_model.Location, error) {
	l.ID = strings.TrimSpace(l.ID)
	if l.ID == "" {
		return l, fmt.Errorf("id is required")
	}
	if l.Latitude == nil || !finite(*l.Latitude) || *l.Latitude < -90 || *l.Latitude > 90 {
		return l, fmt.Errorf("latitude must be between -90 and 90")
	}
	if l.Longitude == nil || !finite(*l.Longitude) || *l.Longitude < -180 || *l.Longitude > 180 {
		return l, fmt.Errorf("longitude must be between -180 and 180")
	}
	radius := DefaultRadiusKm
	if l.RadiusKm != nil && *l.RadiusKm != 0 {
		radius = *l.RadiusKm
	}
	if !finite(radius) || radius <= 0 || radius > MaxRadiusKm {
		return l, fmt.Errorf("radius_km must be greater than 0 and at most %g", MaxRadiusKm)
	}
	l.RadiusKm = &radius
	return l, nil
}

func finite(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}

// score weighs every event as one plus its expected attendance.
func score(stats congestion_model.AreaStats) float64 {
	return float64(stats.EventsCount + stats.ExpectedAttendance)
}

// congestionFactor maps a score to [0.1, 1.0] with a saturating curve: 0.1 + 0.9·(1 − e^(−score/k)),
// rounded to 2 decimals.
func congestionFactor(score, k float64) float64 {
	if score <= 0 {
		return minFactor
	}
	f := minFactor + (maxFactor-minFactor)*(1-math.Exp(-score/k))
	f = math.Round(f*100) / 100
	return math.Min(maxFactor, math.Max(minFactor, f))
}

func levelFor(factor float64) congestion_model.Level {
	switch {
	case factor >= highThreshold:
		return congestion_model.LevelHigh
	case factor >= mediumThreshold:
		return congestion_model.LevelMedium
	default:
		return congestion_model.LevelLow
	}
}
