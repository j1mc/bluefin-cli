package cmd

import (
	"strings"
	"testing"
)

// Individual probes are covered in internal/doctor; this only checks the
// cmd-level wiring renders a report.
func TestDoctorReport_ProducesOutput(t *testing.T) {
	report, failures := doctorReport()
	if report == "" {
		t.Error("expected non-empty report text")
	}
	if failures < 0 {
		t.Error("failures count should never be negative")
	}
	if failures == 0 && !strings.Contains(report, "All checks passed") {
		t.Error("zero failures should render the all-clear message")
	}
}
