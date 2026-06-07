package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSchedulerDefaultAuctionCheckIntervalKeepsEndAnimationResponsive(t *testing.T) {
	scheduler := NewScheduler(nil)

	require.Equal(t, 200*time.Millisecond, scheduler.checkInterval)
	require.Equal(t, 5*time.Second, scheduler.timeSyncInterval)
}
