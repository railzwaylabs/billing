package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/railzwaylabs/billing/internal/monitoring/domain"
)

type Config struct {
	URL     string
	Timeout time.Duration
}

type Client struct {
	queryURL string
	http     *http.Client
}

func NewClient(config Config) (*Client, error) {
	base, err := url.Parse(config.URL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return nil, fmt.Errorf("invalid Prometheus URL")
	}
	base.Path = "/api/v1/query_range"
	base.RawQuery, base.Fragment = "", ""
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	return &Client{queryURL: base.String(), http: &http.Client{Timeout: config.Timeout}}, nil
}

func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]domain.Sample, error) {
	requestURL, _ := url.Parse(c.queryURL)
	params := requestURL.Query()
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(start.Unix(), 10))
	params.Set("end", strconv.FormatInt(end.Unix(), 10))
	params.Set("step", strconv.FormatInt(int64(step.Seconds()), 10))
	requestURL.RawQuery = params.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("query Prometheus: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query Prometheus: status %d", response.StatusCode)
	}
	var payload struct {
		Status string `json:"status"`
		Error  string `json:"error"`
		Data   struct {
			Result []struct {
				Values [][]json.RawMessage `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode Prometheus response: %w", err)
	}
	if payload.Status != "success" {
		return nil, fmt.Errorf("query Prometheus: %s", payload.Error)
	}
	if len(payload.Data.Result) == 0 {
		return []domain.Sample{}, nil
	}
	samples := make([]domain.Sample, 0, len(payload.Data.Result[0].Values))
	for _, value := range payload.Data.Result[0].Values {
		if len(value) != 2 {
			return nil, fmt.Errorf("invalid Prometheus sample")
		}
		var timestamp float64
		var encoded string
		if err := json.Unmarshal(value[0], &timestamp); err != nil {
			return nil, fmt.Errorf("decode Prometheus timestamp: %w", err)
		}
		if err := json.Unmarshal(value[1], &encoded); err != nil {
			return nil, fmt.Errorf("decode Prometheus value: %w", err)
		}
		number, err := strconv.ParseFloat(encoded, 64)
		if err != nil {
			return nil, fmt.Errorf("parse Prometheus value: %w", err)
		}
		samples = append(samples, domain.Sample{Timestamp: int64(timestamp), Value: number})
	}
	return samples, nil
}
