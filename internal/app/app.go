package app

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"humidity-reminder/internal/config"
	"humidity-reminder/internal/humidity"
	"humidity-reminder/internal/mailgun"
	"humidity-reminder/internal/state"
	"humidity-reminder/internal/weather"

	"net/http"
)

const overnightPeriodsNeeded = 7

// Run executes the application with the provided configuration.
func Run(ctx context.Context, cfg *config.Config) error {
	httpClient := &http.Client{Timeout: cfg.Weather.Timeout}

	weatherClient := weather.NewClient(httpClient, cfg.Weather.UserAgent)
	periods, err := weatherClient.ForecastPeriods(ctx, cfg.Latitude, cfg.Longitude)
	if err != nil {
		return fmt.Errorf("fetch forecast: %w", err)
	}

	lows, err := selectNighttimeLows(periods, overnightPeriodsNeeded)
	if err != nil {
		return err
	}

	minLow := minimum(lows)
	recommended := humidity.RecommendedIndoorHumidity(minLow)
	roundedRecommendation := roundDownToNearestFive(recommended)

	store, err := state.NewStore(cfg.StateDir)
	if err != nil {
		return err
	}
	currentState, err := store.Load()
	if err != nil {
		// Log the error but proceed with an empty state to recover from corruption
		log.Printf("warning: could not load previous state (starting fresh): %v", err)
		currentState = state.State{}
	}
	changed := currentState.LastRecommendation == nil || *currentState.LastRecommendation != roundedRecommendation

	if changed {
		log.Printf("humidity recommendation changed to %d%% (minimum overnight low %.1f°F)", roundedRecommendation, minLow)
		mailgunClient := mailgun.NewClient(cfg.Mailgun.Domain, cfg.Mailgun.APIKey, httpClient)

		subject := fmt.Sprintf("Indoor Humidity Update: %d%%", roundedRecommendation)
		body := buildEmailBody(roundedRecommendation, minLow, lows, currentState.LastRecommendation)
		if err := mailgunClient.Send(ctx, cfg.Mailgun.From, cfg.Mailgun.To, subject, body); err != nil {
			return fmt.Errorf("send mail notification: %w", err)
		}
	} else {
		log.Printf("humidity recommendation unchanged at %d%%", roundedRecommendation)
	}

	now := time.Now().UTC()
	rec := roundedRecommendation

	currentState.LastRecommendation = &rec
	currentState.LastRun = &now

	if err := store.Save(currentState); err != nil {
		return err
	}
	return nil
}

func selectNighttimeLows(periods []weather.Period, count int) ([]float64, error) {
	lows := make([]float64, 0, count)
	for _, p := range periods {
		if !p.IsDaytime {
			tempF, err := toFahrenheit(p.Temperature, p.TemperatureUnit)
			if err != nil {
				return nil, err
			}
			lows = append(lows, tempF)
			if len(lows) == count {
				break
			}
		}
	}

	if len(lows) < count {
		return nil, fmt.Errorf("forecast only returned %d overnight periods; need %d", len(lows), count)
	}

	return lows, nil
}

func toFahrenheit(value int, unit string) (float64, error) {
	switch strings.ToUpper(unit) {
	case "F":
		return float64(value), nil
	case "C":
		return float64(value)*9/5 + 32, nil
	default:
		return 0, fmt.Errorf("unsupported temperature unit %q", unit)
	}
}

func minimum(values []float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func roundDownToNearestFive(value int) int {
	if value < 0 {
		return value
	}
	return (value / 5) * 5
}

func buildEmailBody(newRecommendation int, minLow float64, lows []float64, previous *int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("The recommended indoor humidity is now %d%%.\n\n", newRecommendation))
	sb.WriteString(fmt.Sprintf("Minimum overnight low for the next %d nights: %.1f°F.\n", len(lows), minLow))
	sb.WriteString(fmt.Sprintf("Forecast overnight lows: %s.\n", formatLows(lows)))
	if previous != nil {
		sb.WriteString(fmt.Sprintf("Previous recommendation: %d%%.\n", *previous))
	} else {
		sb.WriteString("This is the first stored recommendation.\n")
	}
	sb.WriteString("\nThis message was generated automatically by humidity-reminder.\n")
	return sb.String()
}

func formatLows(lows []float64) string {
	parts := make([]string, len(lows))
	for i, low := range lows {
		parts[i] = fmt.Sprintf("%.0f°F", math.Round(low))
	}
	return strings.Join(parts, ", ")
}
