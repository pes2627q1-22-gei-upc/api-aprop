package repositories

import (
	"errors"
	"fmt"
	"sync"
	"encoding/json"
	"net/http"
	. "api-aprop/internal/models"
	"github.com/joho/godotenv"
	"os"
	"bytes"
	"strings"
	"time"
	"io"
)

var (
	ErrEventNotFound = errors.New("event not found")
	ErrInvalidEvent  = errors.New("invalid event")
)

type EventStore interface {
	Create(title string, description string) (Event, error)
	EventCreate(e Event) (Event, error)
	Get(id int) (Event, error)
	List() []Event
}

type memoryStore struct {
	mu     sync.RWMutex
	events map[int]Event
	nextID int
}

func NewMemoryStore() *memoryStore {
	return &memoryStore{events: make(map[int]Event), nextID: 1}
}

func (s *memoryStore) Create(title string, description string) (Event, error) {
	if title == "" {
		return Event{}, fmt.Errorf("create event: %w", ErrInvalidEvent)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t := Event{ID: s.nextID, Title: title, Description: description}
	s.events[t.ID] = t
	s.nextID++
	return t, nil
}

func (s *memoryStore) EventCreate(e Event) (Event, error){
	if _, exists := s.events[e.ID]; !exists {
		e.ID=s.nextID
		s.nextID++
	}
	s.events[e.ID]=e
	return e, nil
}

func (s *memoryStore) Get(id int) (Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.events[id]
	if !ok {
		return Event{}, fmt.Errorf("get event %d: %w", id, ErrEventNotFound)
	}
	return t, nil
}

func (s *memoryStore) List() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Event, 0, len(s.events))
	for _, t := range s.events {
		out = append(out, t)
	}
	return out
}

func LoadOfficialEvents() {
	Events := NewMemoryStore()
	
	godotenv.Load(".env.example")
	apptoken := os.Getenv("APP_TOKEN")

	jsonBody := []byte(`{"query": "SELECT *", "order":"data_inici DESC" "page":{"pageNumber":1,"pageSize":10}"}`)
 	bodyReader := bytes.NewReader(jsonBody)

 	req, err := http.NewRequest(http.MethodPost, "https://analisi.transparenciacatalunya.cat/api/v3/views/rhpv-yr4f/query.json", bodyReader)
	if err==nil{
		req.Header.Set("X-App-Token", apptoken)
		req.Header.Set("Content-Type", "application/json")
	} else {
		panic(err)
	}

	client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

	var body []map[string]any
	bodyBytes, err2 := io.ReadAll(resp.Body)
	if err2 != nil{
		panic(err2)
	}
	err3 := json.Unmarshal([]byte(bodyBytes), &body)
	if err3 != nil {
		panic(err2)
	}

	for _, val := range body{
		var e Event
		var tmp string
		e.Code, _=val["codi"].(int64)
		e.Title, _=val["denominaci"].(string)
		e.Subtitle, _=val["subt_tol"].(string)
		e.Schedule, _=val["horari"].(string)
		e.Tickets, _=val["entrades"].(string)
		e.URL, _=val["url"].(string)
		e.TicketsURL, _=val["linkbotoentrades"].(string)
		e.Description, _=val["descripcio"].(string)
		e.Free, _=val["gratuita"].(bool)
		e.Permanent, _=val["permanent"].(bool)
		tmp, _ = val["enlla_os"].(string)
		e.RelatedURLs=strings.Split(tmp, ",")
		tmp, _ = val["enllac1_nom"].(string)
		e.URLnames=append(e.URLnames, tmp)
		tmp, _ = val["enllac2_nom"].(string)
		e.URLnames=append(e.URLnames, tmp)
		tmp, _ = val["enllac3_nom"].(string)
		e.URLnames=append(e.URLnames, tmp)
		tmp, _ = val["modalitat"].(string)
		e.Online=(tmp=="Virtual")
		tmp, _ = val["imatges"].(string)
		e.Images=strings.Split(tmp, ",")
		e.Videos, _=val["v_deos"].(string)
		e.Address, _=val["adre_a"].(string)
		e.PostalCode, _=val["codi_postal"].(string)
		e.Email, _=val["email"].(string)
		e.Place, _=val["espai"].(string)
		e.Latitude, _=val["latitud"].(float32)
		e.Longitude, _=val["longitud"].(float32)
		e.Phone, _=val["tel_fon"].(string)
		e.ActivityURL, _=val["urlactivitat"].(string)
		tmp, _ = val["data_inici"].(string)
		e.StartDate, err=time.Parse("2020-05-31T00:00:00.000", tmp)
		tmp, _ = val["data_fi"].(string)
		e.EndDate, err=time.Parse("2020-05-31T00:00:00.000", tmp)

		Events.EventCreate(e)
	}
}