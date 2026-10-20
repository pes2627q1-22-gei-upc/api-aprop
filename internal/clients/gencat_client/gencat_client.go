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
	"io"

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
	Page queryPage	`json:"page"`
}

func (c *GencatClient) FetchEvents(ctx context.Context, pageNumber int, pageSize int) ([]event_model.Event, error) {
	jsonBody, err := json.Marshal(queryRequest{
		Query: "SELECT * WHERE data_fi IS NULL OR data_fi > '2026-10-18T00:00:00' ORDER BY CASE WHEN data_fi IS NULL THEN 1 ELSE 0 END ASC, data_fi ASC",
		Page: queryPage{
			PageNumber: pageNumber,
			PageSize: pageSize,
		},
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
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf(
				"fetch events: %v: status=%d, body=%s",
				ErrUnexpectedStatus,
				resp.StatusCode,
				string(body),
			)
		}
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

	if v, ok := val["longitud"].(string); ok {
		f, _ := strconv.ParseFloat(v, 32)
		e.Longitude = float32(f)
	}
	if v, ok := val["latitud"].(string); ok {
		f, _ := strconv.ParseFloat(v, 32)
		e.Latitude = float32(f)
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

	e.Modality, _ = val["modalitat"].(string)
	if e.Modality == "Presencial"{
		e.Modality = "Face-to-Face"
	} else if e.Modality == "Virtual"{
		e.Modality = "Online"
	} else if e.Modality == "Presencial o Virtual"{
		e.Modality = "Online and Face-to-Face"
	}
	e.Place, _ = val["espai"].(string)
	e.PostalCode, _ = val["codi_postal"].(string)
	e.Address, _ = val["adre_a"].(string)
	e.Municipality, _ = val["municipi"].(string)
	e.Region, _ = val["comarca"].(string)

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
