package combined_handler

import (
	"fmt"
	"time"
)

// checkDataAge returns an error when maxDataAge > 0 and the data is older than that limit.
func checkDataAge(dataTime time.Time, maxDataAge time.Duration) error {
	if maxDataAge <= 0 {
		return nil
	}
	age := time.Since(dataTime)
	if age > maxDataAge {
		return fmt.Errorf("data age %s exceeds maximum allowed %s", age.Round(time.Second), maxDataAge)
	}
	return nil
}
