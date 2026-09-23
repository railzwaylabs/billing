package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/railzwaylabs/billing/internal/logviewer/domain"
)

type Config struct {
	Provider     string
	URL          string
	TenantID     string
	Username     string
	Password     string
	Timeout      time.Duration
	ServiceLabel string
}

type Client struct {
	queryURL     string
	tenantID     string
	username     string
	password     string
	serviceLabel string
	http         *http.Client
}

func NewClient(config Config) (*Client, error) {
	base, err := url.Parse(config.URL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return nil, fmt.Errorf("invalid Loki URL")
	}

	base.Path = "/loki/api/v1/query_range"
	base.RawQuery, base.Fragment = "", ""
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}

	if config.ServiceLabel == "" {
		config.ServiceLabel = "billing_service"
	}

	if !validLabelName(config.ServiceLabel) {
		return nil, fmt.Errorf("invalid Loki service label")
	}

	return &Client{
		queryURL: base.String(), tenantID: config.TenantID,
		username: config.Username, password: config.Password,
		serviceLabel: config.ServiceLabel, http: &http.Client{Timeout: config.Timeout},
	}, nil
}

func (c *Client) Query(ctx context.Context, query domain.Query) (domain.Page, error) {
	end := query.End
	if query.Cursor != "" {
		nanoseconds, err := strconv.ParseInt(query.Cursor, 10, 64)
		if err != nil {
			return domain.Page{}, fmt.Errorf("invalid cursor: %w", err)
		}

		end = time.Unix(0, nanoseconds-1).UTC()
	}

	logQL := fmt.Sprintf(`{%s=%q}`, c.serviceLabel, query.Service)
	if query.Level != "" {
		logQL += fmt.Sprintf(` | json | level=%q`, query.Level)
	}

	if query.Search != "" {
		logQL += " |= " + strconv.Quote(query.Search)
	}

	requestURL, _ := url.Parse(c.queryURL)
	parameters := requestURL.Query()
	parameters.Set("query", logQL)
	parameters.Set("start", strconv.FormatInt(query.Start.UnixNano(), 10))
	parameters.Set("end", strconv.FormatInt(end.UnixNano(), 10))
	parameters.Set("limit", strconv.Itoa(query.Limit))
	parameters.Set("direction", "backward")
	requestURL.RawQuery = parameters.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return domain.Page{}, err
	}

	if c.tenantID != "" {
		request.Header.Set("X-Scope-OrgID", c.tenantID)
	}

	if c.username != "" {
		request.SetBasicAuth(c.username, c.password)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return domain.Page{}, fmt.Errorf("query Loki: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return domain.Page{}, fmt.Errorf("query Loki: status %d", response.StatusCode)
	}

	var payload struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Stream map[string]string `json:"stream"`
				Values [][]string        `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return domain.Page{}, fmt.Errorf("decode Loki response: %w", err)
	}

	if payload.Status != "success" {
		return domain.Page{}, fmt.Errorf("query Loki failed")
	}

	entries := make([]domain.Entry, 0)
	for _, stream := range payload.Data.Result {
		for _, value := range stream.Values {
			if len(value) != 2 {
				continue
			}

			nanoseconds, err := strconv.ParseInt(value[0], 10, 64)
			if err != nil {
				continue
			}

			entries = append(entries, decodeEntry(query.Service, time.Unix(0, nanoseconds).UTC(), value[1]))
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Timestamp.After(entries[j].Timestamp) })
	if len(entries) > query.Limit {
		entries = entries[:query.Limit]
	}

	page := domain.Page{Entries: entries}
	if len(entries) == query.Limit {
		page.NextCursor = strconv.FormatInt(entries[len(entries)-1].Timestamp.UnixNano(), 10)
	}

	return page, nil
}

func decodeEntry(service string, timestamp time.Time, line string) domain.Entry {
	entry := domain.Entry{Timestamp: timestamp, Service: service, Message: line}
	fields := make(map[string]any)
	if json.Unmarshal([]byte(line), &fields) != nil {
		return entry
	}

	if value, ok := fields["level"].(string); ok {
		entry.Level = value
		delete(fields, "level")
	}

	for _, key := range []string{"msg", "message"} {
		if value, ok := fields[key].(string); ok {
			entry.Message = value
			delete(fields, key)
			break
		}
	}

	delete(fields, "ts")
	delete(fields, "timestamp")
	if len(fields) > 0 {
		entry.Fields = fields
	}

	return entry
}

func validLabelName(value string) bool {
	for index, character := range value {
		if character == '_' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' {
			continue
		}

		if index > 0 && character >= '0' && character <= '9' {
			continue
		}

		return false
	}
	return value != ""
}

type DisabledProvider struct{}

func (DisabledProvider) Query(context.Context, domain.Query) (domain.Page, error) {
	return domain.Page{}, fmt.Errorf("log provider is disabled")
}

func NewProvider(config Config) (domain.Provider, error) {
	switch strings.ToLower(strings.TrimSpace(config.Provider)) {
	case "", "disabled":
		return DisabledProvider{}, nil
	case "loki":
		return NewClient(config)
	default:
		return nil, fmt.Errorf("unsupported log provider %q", config.Provider)
	}
}
