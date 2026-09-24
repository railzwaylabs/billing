package prometheus

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestQueryRangeEncodesParametersAndParsesSamples(t *testing.T) {
	client, err := NewClient(Config{URL: "https://prometheus.example", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/api/v1/query_range" {
			t.Fatalf("path = %s", request.URL.Path)
		}
		if request.URL.Query().Get("query") != "fixed query" || request.URL.Query().Get("step") != "3600" {
			t.Fatalf("query parameters = %s", request.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"resultType":"matrix","result":[{"values":[[1790146800,"NaN"],[1790150400,"1.25"]]}]}}`)),
			Request:    request,
		}, nil
	})
	samples, err := client.QueryRange(context.Background(), "fixed query", time.Unix(1790146800, 0), time.Unix(1790150400, 0), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 1 || samples[0].Timestamp != 1790150400 || samples[0].Value != 1.25 {
		t.Fatalf("samples = %#v", samples)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestNewClientRejectsInvalidURL(t *testing.T) {
	if _, err := NewClient(Config{URL: "file:///tmp/prometheus"}); err == nil {
		t.Fatal("expected invalid URL error")
	}
}
