package talaia_client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"api-aprop/internal/models/event_model"
)

const talaiaDateLayout = "2006-01-02T15:04:05.000"

type TalaiaClient struct {
	httpClient *http.Client
	baseURL    string
	appToken   string
}

func NewTalaiaClient(httpClient *http.Client, baseURL string, appToken string) *TalaiaClient {
	return &TalaiaClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		appToken:   appToken,
	}
}

func (c *TalaiaClient) FetchEvents(ctx context.Context, from time.Time, until time.Time) (map[string]event_model.Event, error) {
	url := fmt.Sprintf("%s/esdeveniments?des_de=%s&fins_a=%s", c.baseURL, from.Format("2006-01-02"), until.Format("2006-01-02"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build talaia request: %w", err)
	}
	req.Header.Set("X-API-Key", c.appToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call talaia api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf(
			"fetch astronomical events: %v: status=%d, body=%s",
			ErrUnexpectedStatus,
			resp.StatusCode,
			string(body),
		)
		return nil, fmt.Errorf("fetch astronomical events: %w: %d", ErrUnexpectedStatus, resp.StatusCode)
	}

	var response struct {
		Data []map[string]any `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode talaia response: %w", err)
	}
	body := response.Data

	events := make(map[string]event_model.Event)
	for _, val := range body {
		k, v := mapEvent(val)
		events[k] = v
	}
	return events, nil
}

func mapEvent(val map[string]any) (string, event_model.Event) {
	var e event_model.Event
	var tmp string
	var tmpfloat float64

	id, _ := val["id"].(string)

	e.Title, _ = val["nom"].(string)
	tmp, _ = val["inici"].(string)
	e.StartDate, _ = time.Parse("2006-01-02T15:04:05.000Z", tmp)
	tmp, _ = val["fi"].(string)
	e.EndDate, _ = time.Parse("2006-01-02T15:04:05.000Z", tmp)
	e.Description, _ = val["descripcio"].(string)

	ubicacio, _ := val["ubicacio"].(map[string]any)
	e.Place, _ = ubicacio["nom"].(string)
	tmpfloat, _ = ubicacio["latitud"].(float64)
	e.Latitude = float32(tmpfloat)
	tmpfloat, _ = ubicacio["longitude"].(float64)
	e.Longitude = float32(tmpfloat)

	tmp, _ = val["tipus"].(string)
	e.Categories = []string{"talaia:ambits/astronomic", fmt.Sprintf("talaia:categories/%s", tmp)}

	tmp, ok := val["maxim"].(string)
	if ok {
		maxim, _ := time.Parse("2006-01-02T15:04:05.000Z", tmp)
		e.Schedule = fmt.Sprintf("Moment màxim: %d de %s %d, %s UTC", maxim.Day(), toMonth(int(maxim.Month())), maxim.Year(), toTime(maxim.Hour(), maxim.Minute()))
	}

	return id, e
}

func toMonth(m int) string {
	switch m {
	case 1:
		return "Gener"
	case 2:
		return "Febrer"
	case 3:
		return "Març"
	case 4:
		return "Abril"
	case 5:
		return "Maig"
	case 6:
		return "Juny"
	case 7:
		return "Juliol"
	case 8:
		return "Agost"
	case 9:
		return "Setembre"
	case 10:
		return "Octubre"
	case 11:
		return "Novembre"
	case 12:
		return "Desembre"
	}
	return "Error"
}

func toTime(h int, m int) string {
	return fmt.Sprintf("%02d:%02d", h, m)
}
