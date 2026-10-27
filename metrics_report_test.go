package main

import (
	"testing"
	"time"
)

func TestNeedsReminderModelsDeadlineDecision(t *testing.T) {
	now := time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		item LearnerDeadline
		want bool
	}{
		{"open before due date", LearnerDeadline{Course: "math", Due: now.Add(time.Hour)}, true},
		{"completed work stays quiet", LearnerDeadline{Course: "math", Due: now.Add(time.Hour), Done: true}, false},
		{"past due date is not an upcoming reminder", LearnerDeadline{Course: "math", Due: now.Add(-time.Hour)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NeedsReminder(tc.item, now); got != tc.want {
				t.Fatalf("NeedsReminder() = %t, want %t", got, tc.want)
			}
		})
	}
}
