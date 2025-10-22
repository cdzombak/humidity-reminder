package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.weather.gov"

// Period represents a single forecast period returned by weather.gov.
type Period struct {
	Name            string
	StartTime       time.Time
	IsDaytime       bool
	Temperature     int
	TemperatureUnit string
}

// Client retrieves forecast data from the weather.gov API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

// Option configures the weather client.
type Option func(*Client)

// WithBaseURL overrides the default weather.gov API base URL. Primarily used for tests.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// NewClient constructs a Client with the supplied HTTP client and User-Agent.
// If httpClient is nil, http.DefaultClient is used.
func NewClient(httpClient *http.Client, userAgent string, opts ...Option) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	client := &Client{
		baseURL:    defaultBaseURL,
		httpClient: httpClient,
		userAgent:  userAgent,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// ForecastPeriods returns the forecast periods for the provided coordinates.
func (c *Client) ForecastPeriods(ctx context.Context, lat, lon float64) ([]Period, error) {
	forecastURL, err := c.resolveForecastURL(ctx, lat, lon)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, forecastURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create forecast request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do forecast request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forecast request failed: %s", resp.Status)
	}

	var fr forecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&fr); err != nil {
		return nil, fmt.Errorf("decode forecast: %w", err)
	}

	if len(fr.Properties.Periods) == 0 {
		return nil, errors.New("forecast response missing periods")
	}

	periods := make([]Period, 0, len(fr.Properties.Periods))
	for _, p := range fr.Properties.Periods {
		start, err := time.Parse(time.RFC3339, p.StartTime)
		if err != nil {
			return nil, fmt.Errorf("parse period start time: %w", err)
		}
		periods = append(periods, Period{
			Name:            p.Name,
			StartTime:       start,
			IsDaytime:       p.IsDaytime,
			Temperature:     p.Temperature,
			TemperatureUnit: p.TemperatureUnit,
		})
	}

	return periods, nil
}

func (c *Client) resolveForecastURL(ctx context.Context, lat, lon float64) (string, error) {
	latStr := strconv.FormatFloat(lat, 'f', 4, 64)
	lonStr := strconv.FormatFloat(lon, 'f', 4, 64)
	url := fmt.Sprintf("%s/points/%s,%s", c.baseURL, latStr, lonStr)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create points request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do points request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("points request failed: %s", resp.Status)
	}

	var pr pointsResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return "", fmt.Errorf("decode points response: %w", err)
	}
	if pr.Properties.Forecast == "" {
		return "", errors.New("points response missing forecast URL")
	}

	return pr.Properties.Forecast, nil
}

type pointsResponse struct {
	Properties struct {
		Forecast string `json:"forecast"`
	} `json:"properties"`
}

type forecastResponse struct {
	Properties struct {
		Periods []forecastPeriod `json:"periods"`
	} `json:"properties"`
}

type forecastPeriod struct {
	Name            string `json:"name"`
	StartTime       string `json:"startTime"`
	IsDaytime       bool   `json:"isDaytime"`
	Temperature     int    `json:"temperature"`
	TemperatureUnit string `json:"temperatureUnit"`
}
