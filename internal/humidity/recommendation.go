package humidity

import "github.com/cdzombak/libwx"

// RecommendedIndoorHumidity returns the recommended indoor relative humidity (percentage)
// for the provided outdoor temperature in degrees Fahrenheit.
func RecommendedIndoorHumidity(outdoorTempF float64) int {
	return int(libwx.IndoorHumidityRecommendationF(libwx.TempF(outdoorTempF)))
}
