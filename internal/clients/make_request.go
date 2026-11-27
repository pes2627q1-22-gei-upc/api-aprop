package clients

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

func MakeRequest(ctx context.Context, httpClient *http.Client, method string, url string, APIKeyFieldName string, appToken string, reader io.Reader, service_name string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("build %s request: %w", service_name, err)
	}
	req.Header.Set(APIKeyFieldName, appToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call %s api: %w", service_name, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf(
			"fetch %s events: %v: status=%d, body=%s",
			service_name,
			ErrUnexpectedStatus,
			resp.StatusCode,
			string(body),
		)
		return nil, fmt.Errorf("fetch %s events: %w: %d", service_name, ErrUnexpectedStatus, resp.StatusCode)
	}

	return resp, err
}
