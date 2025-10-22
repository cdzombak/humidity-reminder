package weather_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"humidity-reminder/internal/weather"
)

func TestForecastPeriods(t *testing.T) {
	pointsRequested := false
	forecastRequested := false

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/points/41.1234,-81.5679":
			pointsRequested = true
			if ua := r.Header.Get("User-Agent"); ua != "test-agent" {
				t.Fatalf("unexpected User-Agent for points request: %q", ua)
			}
			resp := map[string]any{
				"properties": map[string]any{
					"forecast": server.URL + "/forecast",
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case "/forecast":
			forecastRequested = true
			if ua := r.Header.Get("User-Agent"); ua != "test-agent" {
				t.Fatalf("unexpected User-Agent for forecast request: %q", ua)
			}
			resp := map[string]any{
				"properties": map[string]any{
					"periods": []map[string]any{
						{
							"name":            "Tonight",
							"startTime":       "2024-11-18T18:00:00-05:00",
							"isDaytime":       false,
							"temperature":     35,
							"temperatureUnit": "F",
						},
						{
							"name":            "Monday",
							"startTime":       "2024-11-19T06:00:00-05:00",
							"isDaytime":       true,
							"temperature":     45,
							"temperatureUnit": "F",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := weather.NewClient(server.Client(), "test-agent", weather.WithBaseURL(server.URL))

	periods, err := client.ForecastPeriods(context.Background(), 41.1234, -81.5679)
	if err != nil {
		t.Fatalf("ForecastPeriods returned error: %v", err)
	}

	if !pointsRequested {
		t.Fatal("points endpoint was not requested")
	}
	if !forecastRequested {
		t.Fatal("forecast endpoint was not requested")
	}

	if len(periods) != 2 {
		t.Fatalf("expected 2 periods, got %d", len(periods))
	}

	if periods[0].Name != "Tonight" || periods[0].IsDaytime {
		t.Fatalf("unexpected first period: %+v", periods[0])
	}
	if periods[0].Temperature != 35 || periods[0].TemperatureUnit != "F" {
		t.Fatalf("unexpected temperature data: %+v", periods[0])
	}
	expectedStart := time.Date(2024, 11, 18, 18, 0, 0, 0, time.FixedZone("", -5*3600))
	if !periods[0].StartTime.Equal(expectedStart) {
		t.Fatalf("unexpected start time: %v", periods[0].StartTime)
	}
}
