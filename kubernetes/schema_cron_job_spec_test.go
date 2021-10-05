package kubernetes

import (
	"testing"
)

// Test Flatteners
func TestCronJobFormatting(t *testing.T) {

	validator := validateCronExpression()
    _, es := validator("CRON_TZ=UTC 30 04 * * *","Should parse out UTC")
	if len(es) != 0 {
		t.Errorf("Failed to parse CRON_TZ spec. #{err}")
	}
	_, es = validator("TZ=UTC 30 04 * * *","Should parse out UTC")
	if len(es) != 0 {
		t.Errorf("Failed to parse TZ spec. #{err}")
	}
	_, es = validator("30 04 * * *","Should accurately parse out a traditional cron format")
	if len(es) != 0 {
		t.Errorf("Failed to parse normal spec. #{err}")
	}
}
