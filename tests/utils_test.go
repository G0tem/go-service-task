package tests

import (
	"testing"
	"time"

	"github.com/G0tem/go-service-task/internal"
)

func TestUtilsParseDurationCase1(t *testing.T) {
	if internal.ParseDuration("1h", 0) != 1*time.Hour {
		t.Errorf("Incorrect parse duration %v", "1h")
	}
	if internal.ParseDuration("1m", 0) != 1*time.Minute {
		t.Errorf("Incorrect parse duration %v", "1m")
	}
	if internal.ParseDuration("1s", 0) != 1*time.Second {
		t.Errorf("Incorrect parse duration %v", "1s")
	}
}

func TestUtilsParseDurationCase2(t *testing.T) {
	if internal.ParseDuration("1h 5m", 0) != (1*time.Hour + 5*time.Minute) {
		t.Errorf("Incorrect parse duration %v", "1h 5m")
	}
}

func TestUtilsParseDurationCase3(t *testing.T) {
	value := 1*time.Hour + 5*time.Minute + 6*time.Second
	if internal.ParseDuration("1h 5m 6 s", 0) != value {
		t.Errorf("Incorrect parse duration %v", "1h 5m 6 s")
	}
}
