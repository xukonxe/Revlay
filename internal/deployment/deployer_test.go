package deployment

import (
	"testing"
	"time"
)

func TestGenerateReleaseTimestamp(t *testing.T) {
	timestamp := GenerateReleaseTimestamp()
	
	// Should be 15 characters long (YYYYMMDD-HHMMSS)
	if len(timestamp) != 15 {
		t.Errorf("Expected timestamp length to be 15, got %d", len(timestamp))
	}
	
	// Should be parseable as a time
	_, err := time.Parse("20060102-150405", timestamp)
	if err != nil {
		t.Errorf("Generated timestamp should be parseable: %v", err)
	}
}

func TestReleaseTimestampUniqueness(t *testing.T) {
	// Generate multiple timestamps with reasonable delays
	timestamps := make(map[string]bool)
	for i := 0; i < 3; i++ {
		timestamp := GenerateReleaseTimestamp()
		if timestamps[timestamp] {
			t.Errorf("Timestamp %s was generated twice", timestamp)
		}
		timestamps[timestamp] = true
		
		// Reasonable delay for real-world scenarios
		time.Sleep(time.Second)
	}
}