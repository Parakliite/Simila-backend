package main

import (
	"context"
	"testing"
	"time"
)

func TestRecordDiscoverySyncFailurePausesAfterFiveFailures(t *testing.T) {
	app := newTestApp("")

	err := app.recordDiscoverySyncFailure(context.Background(), "tv", 4)
	if err != errDiscoverySyncPaused {
		t.Fatalf("expected discovery sync to pause, got %v", err)
	}

	state, err := app.models.Discovery.GetDiscoverySyncState(context.Background(), "tv")
	if err != nil {
		t.Fatalf("failed to read sync state: %v", err)
	}
	if state.ConsecutiveFailures != 5 {
		t.Fatalf("expected 5 consecutive failures, got %d", state.ConsecutiveFailures)
	}
	if state.PausedUntil == nil || !state.PausedUntil.After(time.Now()) {
		t.Fatal("expected sync to be paused")
	}
}
