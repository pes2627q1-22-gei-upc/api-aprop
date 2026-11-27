package preference_service

import (
	"context"
	"errors"
	"testing"
	"time"

	"api-aprop/internal/models/event_model"
	"api-aprop/internal/models/preference_model"
)

const validUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

type fakePreferenceRepository struct {
	questions          []preference_model.TestQuestion
	questionsErr       error
	gotLimit           int
	gotExcludedIDs     []int
	hasCompleted       bool
	hasCompleteErr     error
	testStatus         preference_model.TestStatus
	testStatusErr      error
	gotStatusUID       string
	prompted           bool
	markPromptedErr    error
	markPromptedCalls  int
	gotPromptedUID     string
	progress           []preference_model.TestAnswer
	progressErr        error
	savePartialErr     error
	savePartialCalls   int
	lastPartialAnswers []preference_model.TestAnswer

	saveErr        error
	savedUserID    string
	savedDateTaken time.Time
	savedAnswers   []preference_model.TestAnswer
	savedPrefs     []preference_model.UserPreference
	saveCalls      int

	eventCategories    map[int][]int
	eventCategoriesErr error

	allCategories    []event_model.Category
	allCategoriesErr error

	userPreferences    []preference_model.UserPreference
	userPreferencesErr error
}

func (f *fakePreferenceRepository) GetQuestions(_ context.Context, limit int) ([]preference_model.TestQuestion, error) {
	f.gotLimit = limit
	f.gotExcludedIDs = nil
	return f.questions, f.questionsErr
}

func (f *fakePreferenceRepository) GetQuestionsExcluding(_ context.Context, limit int, excludeEventIDs []int) ([]preference_model.TestQuestion, error) {
	f.gotLimit = limit
	f.gotExcludedIDs = append([]int(nil), excludeEventIDs...)
	if f.questionsErr != nil {
		return nil, f.questionsErr
	}
	excluded := make(map[int]struct{}, len(excludeEventIDs))
	for _, id := range excludeEventIDs {
		excluded[id] = struct{}{}
	}
	filtered := make([]preference_model.TestQuestion, 0, len(f.questions))
	for _, q := range f.questions {
		if _, skip := excluded[q.EventID]; skip {
			continue
		}
		filtered = append(filtered, q)
		if len(filtered) >= limit {
			break
		}
	}
	return filtered, nil
}

func (f *fakePreferenceRepository) HasCompletedTest(_ context.Context, _ string) (bool, error) {
	return f.hasCompleted, f.hasCompleteErr
}

func (f *fakePreferenceRepository) GetTestStatus(_ context.Context, userID string) (preference_model.TestStatus, error) {
	f.gotStatusUID = userID
	if f.testStatusErr != nil {
		return preference_model.TestStatus{}, f.testStatusErr
	}
	status := f.testStatus
	status.InProgress = len(f.progress) > 0
	status.AnsweredCount = len(f.progress)
	if f.prompted || status.Completed || status.InProgress {
		status.Prompted = true
	}
	return status, nil
}

func (f *fakePreferenceRepository) MarkTestPrompted(_ context.Context, userID string) error {
	f.markPromptedCalls++
	f.gotPromptedUID = userID
	if f.markPromptedErr != nil {
		return f.markPromptedErr
	}
	f.prompted = true
	return nil
}

func (f *fakePreferenceRepository) GetTestProgress(_ context.Context, _ string) ([]preference_model.TestAnswer, error) {
	if f.progressErr != nil {
		return nil, f.progressErr
	}
	return append([]preference_model.TestAnswer(nil), f.progress...), nil
}

func (f *fakePreferenceRepository) SavePartialProgress(
	_ context.Context,
	_ string,
	_ time.Time,
	answers []preference_model.TestAnswer,
) error {
	f.savePartialCalls++
	f.lastPartialAnswers = append([]preference_model.TestAnswer(nil), answers...)
	if f.savePartialErr != nil {
		return f.savePartialErr
	}
	f.progress = mergeAnswers(f.progress, answers)
	f.prompted = true
	return nil
}

func (f *fakePreferenceRepository) SaveTestAndPreferences(
	_ context.Context,
	userID string,
	dateTaken time.Time,
	answers []preference_model.TestAnswer,
	preferences []preference_model.UserPreference,
) error {
	f.saveCalls++
	f.savedUserID = userID
	f.savedDateTaken = dateTaken
	f.savedAnswers = answers
	f.savedPrefs = preferences
	if f.saveErr != nil {
		return f.saveErr
	}
	// Mirror R1 replace semantics: latest test replaces previous state, clears draft and marks prompted.
	dt := dateTaken
	f.hasCompleted = true
	f.prompted = true
	f.progress = nil
	f.testStatus = preference_model.TestStatus{Completed: true, Prompted: true, DateTaken: &dt}
	f.userPreferences = preferences
	return nil
}

func (f *fakePreferenceRepository) GetEventCategories(_ context.Context, _ []int) (map[int][]int, error) {
	return f.eventCategories, f.eventCategoriesErr
}

func (f *fakePreferenceRepository) ListAllCategories(_ context.Context) ([]event_model.Category, error) {
	return f.allCategories, f.allCategoriesErr
}

func (f *fakePreferenceRepository) GetUserPreferences(_ context.Context, _ string) ([]preference_model.UserPreference, error) {
	return f.userPreferences, f.userPreferencesErr
}

func TestGetQuestions(t *testing.T) {
	repoErr := errors.New("db down")
	tests := []struct {
		name      string
		userID    string
		repo      fakePreferenceRepository
		wantCount int
		wantErr   error
	}{
		{
			name:   "success returns questions",
			userID: validUserID,
			repo: fakePreferenceRepository{
				questions: []preference_model.TestQuestion{
					{EventID: 1, Title: "Concert", Images: "https://example.cat/1.jpg", Categories: []string{"agenda:categories/musica"}},
					{EventID: 2, Title: "Teatre", Images: "https://example.cat/2.jpg", Categories: []string{"agenda:categories/dansa"}},
				},
			},
			wantCount: 2,
		},
		{
			name:      "nil questions returns empty slice",
			userID:    validUserID,
			repo:      fakePreferenceRepository{questions: nil},
			wantCount: 0,
		},
		{
			name:    "invalid user id returns error",
			userID:  "not-a-uuid",
			wantErr: ErrInvalidUserID,
		},
		{
			name:    "repository error is propagated",
			userID:  validUserID,
			repo:    fakePreferenceRepository{questionsErr: repoErr},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewPreferenceService(&tt.repo)
			got, err := s.GetQuestions(context.Background(), tt.userID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("got %d questions, want %d", len(got), tt.wantCount)
			}
			if tt.repo.gotLimit != defaultTestLimit {
				t.Errorf("repo limit = %d, want %d", tt.repo.gotLimit, defaultTestLimit)
			}
		})
	}
}

func TestGetTestStatus(t *testing.T) {
	fixedTime := time.Date(2026, 10, 24, 18, 0, 0, 0, time.UTC)
	repoErr := errors.New("db down")

	tests := []struct {
		name          string
		userID        string
		repo          fakePreferenceRepository
		wantCompleted bool
		wantDateTaken *time.Time
		wantErr       error
	}{
		{
			name:          "not completed returns false and nil date",
			userID:        validUserID,
			repo:          fakePreferenceRepository{testStatus: preference_model.TestStatus{Completed: false, DateTaken: nil}},
			wantCompleted: false,
			wantDateTaken: nil,
		},
		{
			name:   "completed returns true and timestamp",
			userID: "3F2504E0-4F89-11D3-9A0C-0305E82C3301",
			repo: fakePreferenceRepository{
				testStatus: preference_model.TestStatus{Completed: true, DateTaken: &fixedTime},
			},
			wantCompleted: true,
			wantDateTaken: &fixedTime,
		},
		{
			name:    "invalid user id returns ErrInvalidUserID",
			userID:  "not-a-uuid",
			wantErr: ErrInvalidUserID,
		},
		{
			name:    "repository error is propagated",
			userID:  validUserID,
			repo:    fakePreferenceRepository{testStatusErr: repoErr},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewPreferenceService(&tt.repo)
			got, err := s.GetTestStatus(context.Background(), tt.userID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if tt.repo.gotStatusUID != validUserID {
				t.Errorf("normalized userID = %q, want %q", tt.repo.gotStatusUID, validUserID)
			}
			if got.Completed != tt.wantCompleted {
				t.Errorf("completed = %v, want %v", got.Completed, tt.wantCompleted)
			}
			if tt.wantDateTaken == nil {
				if got.DateTaken != nil {
					t.Errorf("dateTaken = %v, want nil", got.DateTaken)
				}
			} else if got.DateTaken == nil || !got.DateTaken.Equal(*tt.wantDateTaken) {
				t.Errorf("dateTaken = %v, want %v", got.DateTaken, *tt.wantDateTaken)
			}
		})
	}
}

func TestR1_SubmitTest_RetakeReplacesPreviousTest(t *testing.T) {
	t1 := time.Date(2026, 10, 24, 18, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 10, 25, 10, 30, 0, 0, time.UTC)
	t3 := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)

	repo := &fakePreferenceRepository{
		allCategories: []event_model.Category{
			{ID: 1, Name: "agenda:categories/musica"},
			{ID: 2, Name: "agenda:categories/teatre"},
			{ID: 3, Name: "agenda:categories/dansa"},
		},
		eventCategories: map[int][]int{
			10: {1},
			20: {2},
			30: {3},
		},
	}

	now := t1
	s := NewPreferenceService(repo)
	s.now = func() time.Time { return now }

	// Before any submission, status is not completed.
	status, err := s.GetTestStatus(context.Background(), validUserID)
	if err != nil {
		t.Fatalf("initial GetTestStatus: %v", err)
	}
	if status.Completed || status.DateTaken != nil {
		t.Fatalf("initial status = %+v, want {Completed:false DateTaken:nil}", status)
	}

	// First submission: likes Música (1.2), dislikes Teatre (0.8), Dansa untouched (1.0).
	res1, err := s.SubmitTest(context.Background(), validUserID, []preference_model.TestAnswer{
		{EventID: 10, Likes: true},
		{EventID: 20, Likes: false},
	})
	if err != nil {
		t.Fatalf("first SubmitTest: %v", err)
	}
	if !res1.DateTaken.Equal(t1) {
		t.Errorf("first dateTaken = %v, want %v", res1.DateTaken, t1)
	}

	// Second submission (retake): dislikes Música (0.8), likes Dansa (1.2), Teatre reset to 1.0.
	now = t2
	retakeAnswers := []preference_model.TestAnswer{
		{EventID: 10, Likes: false},
		{EventID: 30, Likes: true},
	}
	res2, err := s.SubmitTest(context.Background(), validUserID, retakeAnswers)
	if err != nil {
		t.Fatalf("second SubmitTest (retake): %v", err)
	}
	if !res2.DateTaken.Equal(t2) {
		t.Errorf("second dateTaken = %v, want %v", res2.DateTaken, t2)
	}
	if repo.saveCalls != 2 {
		t.Errorf("saveCalls = %d, want 2", repo.saveCalls)
	}
	if len(repo.savedAnswers) != 2 || repo.savedAnswers[0].EventID != 10 || repo.savedAnswers[1].EventID != 30 {
		t.Errorf("savedAnswers = %+v, want replaced answers %+v", repo.savedAnswers, retakeAnswers)
	}

	weights := make(map[int]float32, len(res2.Preferences))
	for _, p := range res2.Preferences {
		weights[p.CategoryID] = p.Weight
	}
	if weights[1] != 0.8 || weights[2] != 1.0 || weights[3] != 1.2 {
		t.Errorf("recomputed weights = %v, want {1:0.8 2:1.0 3:1.2}", weights)
	}

	status, err = s.GetTestStatus(context.Background(), validUserID)
	if err != nil {
		t.Fatalf("GetTestStatus after retake: %v", err)
	}
	if !status.Completed || status.DateTaken == nil || !status.DateTaken.Equal(t2) {
		t.Errorf("status after retake = %+v, want completed at %v", status, t2)
	}

	// Third submission confirms retaking works any number of times.
	now = t3
	res3, err := s.SubmitTest(context.Background(), validUserID, []preference_model.TestAnswer{
		{EventID: 20, Likes: true},
	})
	if err != nil {
		t.Fatalf("third SubmitTest: %v", err)
	}
	if !res3.DateTaken.Equal(t3) || repo.saveCalls != 3 {
		t.Errorf("third submit: dateTaken=%v saveCalls=%d, want %v and 3", res3.DateTaken, repo.saveCalls, t3)
	}
}

func TestSubmitTest(t *testing.T) {
	fixedTime := time.Date(2026, 10, 24, 18, 0, 0, 0, time.UTC)
	repoErr := errors.New("db error")

	defaultCategories := []event_model.Category{
		{ID: 1, Name: "Música"},
		{ID: 2, Name: "Teatre"},
		{ID: 3, Name: "Dansa"},
	}

	tests := []struct {
		name       string
		userID     string
		answers    []preference_model.TestAnswer
		repo       fakePreferenceRepository
		wantErr    error
		wantSave   int
		checkPrefs func(t *testing.T, prefs []preference_model.UserPreference)
	}{
		{
			name:   "successful test calculates weights correctly",
			userID: validUserID,
			answers: []preference_model.TestAnswer{
				{EventID: 10, Likes: true},
				{EventID: 20, Likes: false},
			},
			repo: fakePreferenceRepository{
				allCategories: defaultCategories,
				eventCategories: map[int][]int{
					10: {1}, // Event 10 is Música
					20: {2}, // Event 20 is Teatre
				},
			},
			wantSave: 1,
			checkPrefs: func(t *testing.T, prefs []preference_model.UserPreference) {
				if len(prefs) != 3 {
					t.Fatalf("expected 3 preferences, got %d", len(prefs))
				}
				for _, p := range prefs {
					switch p.CategoryID {
					case 1: // Música (1 like) -> 1.0 + 0.2 = 1.2
						if p.Weight != 1.2 {
							t.Errorf("Música weight = %f, want 1.2", p.Weight)
						}
					case 2: // Teatre (1 dislike) -> 1.0 - 0.2 = 0.8
						if p.Weight != 0.8 {
							t.Errorf("Teatre weight = %f, want 0.8", p.Weight)
						}
					case 3: // Dansa (no answers) -> default 1.0
						if p.Weight != 1.0 {
							t.Errorf("Dansa weight = %f, want 1.0", p.Weight)
						}
					}
				}
			},
		},
		{
			name:    "empty answers returns ErrEmptyAnswers",
			userID:  validUserID,
			answers: []preference_model.TestAnswer{},
			wantErr: ErrEmptyAnswers,
		},
		{
			name:    "too many answers returns ErrTooManyAnswers",
			userID:  validUserID,
			answers: make([]preference_model.TestAnswer, 26),
			wantErr: ErrTooManyAnswers,
		},
		{
			name:   "invalid event id in answers returns ErrInvalidEventID",
			userID: validUserID,
			answers: []preference_model.TestAnswer{
				{EventID: 0, Likes: true},
			},
			wantErr: ErrInvalidEventID,
		},
		{
			name:   "invalid user id returns ErrInvalidUserID",
			userID: "invalid-uuid",
			answers: []preference_model.TestAnswer{
				{EventID: 1, Likes: true},
			},
			wantErr: ErrInvalidUserID,
		},
		{
			name:   "repository error on save is returned",
			userID: validUserID,
			answers: []preference_model.TestAnswer{
				{EventID: 1, Likes: true},
			},
			repo: fakePreferenceRepository{
				allCategories:   defaultCategories,
				eventCategories: map[int][]int{1: {1}},
				saveErr:         repoErr,
			},
			wantErr:  repoErr,
			wantSave: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewPreferenceService(&tt.repo)
			s.now = func() time.Time { return fixedTime }

			res, err := s.SubmitTest(context.Background(), tt.userID, tt.answers)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.repo.saveCalls != tt.wantSave {
				t.Errorf("save calls = %d, want %d", tt.repo.saveCalls, tt.wantSave)
			}
			if tt.wantErr != nil {
				return
			}
			if res.UserID != validUserID {
				t.Errorf("user ID = %s, want %s", res.UserID, validUserID)
			}
			if !res.DateTaken.Equal(fixedTime) {
				t.Errorf("date taken = %v, want %v", res.DateTaken, fixedTime)
			}
			if tt.checkPrefs != nil {
				tt.checkPrefs(t, res.Preferences)
			}
		})
	}
}

func TestGetUserPreferences(t *testing.T) {
	repoErr := errors.New("db error")
	tests := []struct {
		name      string
		userID    string
		repo      fakePreferenceRepository
		wantCount int
		wantErr   error
	}{
		{
			name:   "success returns preferences",
			userID: validUserID,
			repo: fakePreferenceRepository{
				userPreferences: []preference_model.UserPreference{
					{CategoryID: 1, CategoryName: "Música", Weight: 1.5},
				},
			},
			wantCount: 1,
		},
		{
			name:      "nil preferences returns empty slice",
			userID:    validUserID,
			repo:      fakePreferenceRepository{userPreferences: nil},
			wantCount: 0,
		},
		{
			name:    "invalid user id returns ErrInvalidUserID",
			userID:  "bad-uuid",
			wantErr: ErrInvalidUserID,
		},
		{
			name:    "repository error is propagated",
			userID:  validUserID,
			repo:    fakePreferenceRepository{userPreferencesErr: repoErr},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewPreferenceService(&tt.repo)
			prefs, err := s.GetUserPreferences(context.Background(), tt.userID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if len(prefs) != tt.wantCount {
				t.Errorf("got %d preferences, want %d", len(prefs), tt.wantCount)
			}
		})
	}
}

func TestMarkTestPrompted(t *testing.T) {
	repoErr := errors.New("db error")

	t.Run("normalizes user id and marks prompted", func(t *testing.T) {
		repo := &fakePreferenceRepository{}
		s := NewPreferenceService(repo)

		if err := s.MarkTestPrompted(context.Background(), "3F2504E0-4F89-11D3-9A0C-0305E82C3301"); err != nil {
			t.Fatalf("MarkTestPrompted: %v", err)
		}
		if repo.markPromptedCalls != 1 || repo.gotPromptedUID != validUserID || !repo.prompted {
			t.Fatalf("unexpected repo state: calls=%d uid=%q prompted=%v", repo.markPromptedCalls, repo.gotPromptedUID, repo.prompted)
		}
		status, err := s.GetTestStatus(context.Background(), validUserID)
		if err != nil {
			t.Fatalf("GetTestStatus: %v", err)
		}
		if status.Completed || !status.Prompted || status.InProgress {
			t.Errorf("status = %+v, want completed=false prompted=true inProgress=false", status)
		}
	})

	t.Run("invalid user id returns ErrInvalidUserID", func(t *testing.T) {
		s := NewPreferenceService(&fakePreferenceRepository{})
		if err := s.MarkTestPrompted(context.Background(), "bad-uuid"); !errors.Is(err, ErrInvalidUserID) {
			t.Fatalf("error = %v, want %v", err, ErrInvalidUserID)
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		s := NewPreferenceService(&fakePreferenceRepository{markPromptedErr: repoErr})
		if err := s.MarkTestPrompted(context.Background(), validUserID); !errors.Is(err, repoErr) {
			t.Fatalf("error = %v, want %v", err, repoErr)
		}
	})
}

func TestPartialProgress_SaveResumeAndComplete(t *testing.T) {
	t1 := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 21, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)

	repo := &fakePreferenceRepository{
		questions: []preference_model.TestQuestion{
			{EventID: 10, Title: "Concert"},
			{EventID: 20, Title: "Teatre"},
			{EventID: 30, Title: "Dansa"},
		},
		allCategories: []event_model.Category{
			{ID: 1, Name: "Música"},
			{ID: 2, Name: "Teatre"},
			{ID: 3, Name: "Dansa"},
		},
		eventCategories: map[int][]int{
			10: {1},
			20: {2},
			30: {3},
		},
	}

	now := t1
	s := NewPreferenceService(repo)
	s.now = func() time.Time { return now }

	// 1. User answers first question and taps "Més tard" (partial save).
	p1, err := s.SavePartialTest(context.Background(), validUserID, []preference_model.TestAnswer{
		{EventID: 10, Likes: true},
	})
	if err != nil {
		t.Fatalf("first SavePartialTest: %v", err)
	}
	if !p1.Partial || p1.AnsweredCount != 1 || len(p1.Preferences) != 0 {
		t.Fatalf("p1 = %+v, want partial=true answeredCount=1 empty preferences", p1)
	}
	if repo.saveCalls != 0 {
		t.Fatalf("partial save must not overwrite completed tests: saveCalls=%d", repo.saveCalls)
	}

	status, err := s.GetTestStatus(context.Background(), validUserID)
	if err != nil {
		t.Fatalf("GetTestStatus after partial 1: %v", err)
	}
	if status.Completed || !status.Prompted || !status.InProgress || status.AnsweredCount != 1 {
		t.Fatalf("status = %+v, want completed=false prompted=true inProgress=true answeredCount=1", status)
	}

	// 2. Resuming excludes event 10 and caps remaining questions to 25 - 1 = 24.
	remaining, err := s.GetQuestions(context.Background(), validUserID)
	if err != nil {
		t.Fatalf("GetQuestions resume 1: %v", err)
	}
	if repo.gotLimit != defaultTestLimit-1 {
		t.Errorf("limit = %d, want %d", repo.gotLimit, defaultTestLimit-1)
	}
	if len(repo.gotExcludedIDs) != 1 || repo.gotExcludedIDs[0] != 10 {
		t.Errorf("excluded = %v, want [10]", repo.gotExcludedIDs)
	}
	if len(remaining) != 2 || remaining[0].EventID != 20 || remaining[1].EventID != 30 {
		t.Errorf("remaining = %+v, want events 20 and 30", remaining)
	}

	// 3. Second partial save merges event 20 without discarding event 10.
	now = t2
	p2, err := s.SavePartialTest(context.Background(), validUserID, []preference_model.TestAnswer{
		{EventID: 20, Likes: false},
	})
	if err != nil {
		t.Fatalf("second SavePartialTest: %v", err)
	}
	if p2.AnsweredCount != 2 {
		t.Errorf("p2.AnsweredCount = %d, want 2", p2.AnsweredCount)
	}

	// 4. Completing with the final question (event 30) merges 10 + 20 + 30 and computes weights from all 3.
	now = t3
	finalRes, err := s.SubmitTest(context.Background(), validUserID, []preference_model.TestAnswer{
		{EventID: 30, Likes: true},
	})
	if err != nil {
		t.Fatalf("final SubmitTest: %v", err)
	}
	if finalRes.Partial {
		t.Errorf("final SubmitTest must have partial=false")
	}
	if len(repo.savedAnswers) != 3 {
		t.Fatalf("savedAnswers = %+v, want 3 merged answers", repo.savedAnswers)
	}

	weights := make(map[int]float32, len(finalRes.Preferences))
	for _, p := range finalRes.Preferences {
		weights[p.CategoryID] = p.Weight
	}
	if weights[1] != 1.2 || weights[2] != 0.8 || weights[3] != 1.2 {
		t.Errorf("weights = %v, want {1:1.2 2:0.8 3:1.2}", weights)
	}

	status, err = s.GetTestStatus(context.Background(), validUserID)
	if err != nil {
		t.Fatalf("GetTestStatus after completion: %v", err)
	}
	if !status.Completed || !status.Prompted || status.InProgress || status.AnsweredCount != 0 {
		t.Errorf("status after completion = %+v, want completed=true prompted=true inProgress=false answeredCount=0", status)
	}
}

func TestPartialSaveDuringRetake_PreservesPriorCompletedTestUntilFinalSubmit(t *testing.T) {
	t1 := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)

	repo := &fakePreferenceRepository{
		allCategories: []event_model.Category{
			{ID: 1, Name: "Música"},
			{ID: 2, Name: "Teatre"},
		},
		eventCategories: map[int][]int{
			10: {1},
			20: {2},
		},
	}

	now := t1
	s := NewPreferenceService(repo)
	s.now = func() time.Time { return now }

	// Complete initial test: likes Música (1.2).
	if _, err := s.SubmitTest(context.Background(), validUserID, []preference_model.TestAnswer{
		{EventID: 10, Likes: true},
	}); err != nil {
		t.Fatalf("initial SubmitTest: %v", err)
	}

	// Start retake, dislike Música, tap "Més tard" (partial save).
	now = t2
	if _, err := s.SavePartialTest(context.Background(), validUserID, []preference_model.TestAnswer{
		{EventID: 10, Likes: false},
	}); err != nil {
		t.Fatalf("retake SavePartialTest: %v", err)
	}

	// Prior completed test and weights must remain untouched while retake is paused.
	status, err := s.GetTestStatus(context.Background(), validUserID)
	if err != nil {
		t.Fatalf("GetTestStatus during paused retake: %v", err)
	}
	if !status.Completed || !status.InProgress || status.AnsweredCount != 1 || status.DateTaken == nil || !status.DateTaken.Equal(t1) {
		t.Fatalf("status during paused retake = %+v, want completed at t1 + inProgress=true answeredCount=1", status)
	}
	prefs, err := s.GetUserPreferences(context.Background(), validUserID)
	if err != nil {
		t.Fatalf("GetUserPreferences during paused retake: %v", err)
	}
	if prefs[0].Weight != 1.2 {
		t.Errorf("prior preference weight mutated by partial save: got %v, want 1.2", prefs[0].Weight)
	}

	// Completing the retake merges partial dislike of Música + like of Teatre and replaces prior test.
	now = t3
	res, err := s.SubmitTest(context.Background(), validUserID, []preference_model.TestAnswer{
		{EventID: 20, Likes: true},
	})
	if err != nil {
		t.Fatalf("retake completion SubmitTest: %v", err)
	}
	weights := make(map[int]float32, len(res.Preferences))
	for _, p := range res.Preferences {
		weights[p.CategoryID] = p.Weight
	}
	if weights[1] != 0.8 || weights[2] != 1.2 {
		t.Errorf("final retake weights = %v, want {1:0.8 2:1.2}", weights)
	}
	status, _ = s.GetTestStatus(context.Background(), validUserID)
	if !status.Completed || status.InProgress || status.DateTaken == nil || !status.DateTaken.Equal(t3) {
		t.Errorf("status after retake completion = %+v, want completed at t3 and inProgress=false", status)
	}
}

func TestSavePartialTest_ValidationAndTotalLimit(t *testing.T) {
	existing := make([]preference_model.TestAnswer, 20)
	for i := range existing {
		existing[i] = preference_model.TestAnswer{EventID: i + 1, Likes: true}
	}
	incomingTooMany := make([]preference_model.TestAnswer, 6)
	for i := range incomingTooMany {
		incomingTooMany[i] = preference_model.TestAnswer{EventID: 100 + i, Likes: false}
	}

	s := NewPreferenceService(&fakePreferenceRepository{progress: existing})

	if _, err := s.SavePartialTest(context.Background(), validUserID, nil); !errors.Is(err, ErrEmptyAnswers) {
		t.Errorf("empty partial err = %v, want %v", err, ErrEmptyAnswers)
	}
	if _, err := s.SavePartialTest(context.Background(), validUserID, []preference_model.TestAnswer{{EventID: -1, Likes: true}}); !errors.Is(err, ErrInvalidEventID) {
		t.Errorf("invalid event err = %v, want %v", err, ErrInvalidEventID)
	}
	if _, err := s.SavePartialTest(context.Background(), validUserID, incomingTooMany); !errors.Is(err, ErrTooManyAnswers) {
		t.Errorf("merged > 25 err = %v, want %v", err, ErrTooManyAnswers)
	}
	if _, err := s.SubmitTest(context.Background(), validUserID, incomingTooMany); !errors.Is(err, ErrTooManyAnswers) {
		t.Errorf("final submit merged > 25 err = %v, want %v", err, ErrTooManyAnswers)
	}
}
