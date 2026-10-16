package gencat_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"api-aprop/internal/models/event_model"
)

const gencatDateLayout = "2006-01-02T15:04:05.000"

type GencatClient struct {
	httpClient *http.Client
	baseURL    string
	appToken   string
}

func NewGencatClient(httpClient *http.Client, baseURL string, appToken string) *GencatClient {
	return &GencatClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		appToken:   appToken,
	}
}

type queryPage struct {
	PageNumber int `json:"pageNumber"`
	PageSize   int `json:"pageSize"`
}

type queryRequest struct {
	Query string    `json:"query"`
	Page  queryPage `json:"page"`
}

func (c *GencatClient) FetchEvents(ctx context.Context, pageNumber int, pageSize int) ([]event_model.Event, error) {
	jsonBody, err := json.Marshal(queryRequest{
		Query: "SELECT *",
		Page:  queryPage{PageNumber: pageNumber, PageSize: pageSize},
	})
	if err != nil {
		return nil, fmt.Errorf("encode gencat query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("build gencat request: %w", err)
	}
	req.Header.Set("X-App-Token", c.appToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call gencat api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch events: %w: %d", ErrUnexpectedStatus, resp.StatusCode)
	}

	var body []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode gencat response: %w", err)
	}

	events := make([]event_model.Event, 0, len(body))
	for _, val := range body {
		events = append(events, mapEvent(val))
	}
	return events, nil
}

// mapEvent converts a raw row of the gencat dataset into an Event.
func mapEvent(val map[string]any) event_model.Event {
	var e event_model.Event
	var tmp string

	if v, ok := val["codi"].(string); ok {
		e.Code, _ = strconv.ParseInt(v, 10, 64)
	}

	e.Title, _ = val["denominaci"].(string)
	e.Subtitle, _ = val["subt_tol"].(string)
	e.Schedule, _ = val["horari"].(string)
	e.Tickets, _ = val["entrades"].(string)
	e.URL, _ = val["url"].(string)
	e.TicketsURL, _ = val["linkbotoentrades"].(string)
	e.Description, _ = val["descripcio"].(string)
	e.Free, _ = val["gratuita"].(bool)
	e.Permanent, _ = val["permanent"].(bool)

	e.RelatedURLs, _ = val["enlla_os"].(string)

	tmp, _ = val["enllac1_nom"].(string)
	tmp2, _ := val["enllac2_nom"].(string)
	tmp3, _ := val["enllac3_nom"].(string)

	if tmp!=""{
		e.URLnames = tmp
	}
	if tmp2!=""{
		e.URLnames = fmt.Sprintf("%s,%s", e.URLnames, tmp2)
	}
	if tmp3!=""{
		e.URLnames = fmt.Sprintf("%s,%s", e.URLnames, tmp3)
	}

	tmp, _ = val["modalitat"].(string)
	e.Online = tmp == "Virtual"

	e.Images, _ = val["imatges"].(string)

	e.Videos, _ = val["v_deos"].(string)
	e.Email, _ = val["email"].(string)

	e.Phone, _ = val["tel_fon"].(string)
	e.ActivityURL, _ = val["urlactivitat"].(string)

	if tmp, ok := val["data_inici"].(string); ok && tmp != "" {
		startDate, err := time.Parse(gencatDateLayout, tmp)
		if err != nil {
			log.Printf("parse start date %q: %v", tmp, err)
		}
		e.StartDate = startDate
	}

	if tmp, ok := val["data_fi"].(string); ok && tmp != "" {
		endDate, err := time.Parse(gencatDateLayout, tmp)
		if err != nil {
			log.Printf("parse end date %q: %v", tmp, err)
		}
		e.EndDate = endDate
	}

	return e
}
