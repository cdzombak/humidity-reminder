package app

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
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

	medianLow := median(lows)
	recommended := humidity.RecommendedIndoorHumidity(medianLow)
	roundedRecommendation := roundDownToNearestFive(recommended)

	store, err := state.NewStore(cfg.StateDir)
	if err != nil {
		return err
	}

	currentState, err := store.Load()
	if err != nil {
		return err
	}

	changed := currentState.LastRecommendation == nil || *currentState.LastRecommendation != roundedRecommendation

	if changed {
		log.Printf("humidity recommendation changed to %d%% (median overnight low %.1f°F)", roundedRecommendation, medianLow)
		mailgunClient := mailgun.NewClient(cfg.Mailgun.Domain, cfg.Mailgun.APIKey, httpClient)

		subject := fmt.Sprintf("Indoor Humidity Update: %d%%", roundedRecommendation)
		body := buildEmailBody(roundedRecommendation, medianLow, lows, currentState.LastRecommendation)
		if err := mailgunClient.Send(ctx, cfg.Mailgun.From, cfg.Mailgun.To, subject, body); err != nil {
			return fmt.Errorf("send mail notification: %w", err)
		}
	} else {
		log.Printf("humidity recommendation unchanged at %d%%", roundedRecommendation)
	}

	now := time.Now().UTC()
	rec := roundedRecommendation
	saveState := state.State{
		LastRecommendation: &rec,
		LastRun:            &now,
	}

	if err := store.Save(saveState); err != nil {
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

func median(values []float64) float64 {
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	n := len(sorted)
	if n == 0 {
		return math.NaN()
	}
	middle := n / 2
	if n%2 == 1 {
		return sorted[middle]
	}
	return (sorted[middle-1] + sorted[middle]) / 2
}

func roundDownToNearestFive(value int) int {
	if value < 0 {
		return value
	}
	return (value / 5) * 5
}

func buildEmailBody(newRecommendation int, medianLow float64, lows []float64, previous *int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("The recommended indoor humidity is now %d%%.\n\n", newRecommendation))
	sb.WriteString(fmt.Sprintf("Median overnight low for the next %d nights: %.1f°F.\n", len(lows), medianLow))
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
