package subscription

import (
	"strings"
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	t.Parallel()

	date, err := ParseDate("2026/05/01")
	if err != nil {
		t.Fatal(err)
	}
	if got := date.Format(DateLayout); got != "2026-05-01" {
		t.Fatalf("unexpected date: %s", got)
	}
}

func TestBuildReport(t *testing.T) {
	t.Parallel()

	report := BuildReport([]Subscription{
		{
			ID:         2,
			Name:       "Long service",
			TargetDate: time.Date(2026, 5, 10, 12, 0, 0, 0, time.Local),
		},
		{
			ID:         1,
			Name:       "Soon service",
			TargetDate: time.Date(2026, 5, 2, 12, 0, 0, 0, time.Local),
		},
	}, time.Date(2026, 5, 1, 12, 0, 0, 0, time.Local))

	if !strings.Contains(report, "Soon service") {
		t.Fatal("missing subscription name")
	}
	if !strings.Contains(report, "还有 1 天，需要关注") {
		t.Fatal("missing urgent status")
	}
	if strings.Index(report, "Soon service") > strings.Index(report, "Long service") {
		t.Fatal("report is not sorted by target date")
	}
}
