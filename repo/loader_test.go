package repo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	updateDuration = 5 * time.Millisecond
)

func TestRepo_tryUpdate(t *testing.T) {
	repo := &Repo{
		updateInProgressChannel: make(chan chan updateResponse, 1),
	}

	executed := make(chan time.Time, 4)

	// Consumer For Updates
	go func() {
		for {
			res := <-repo.updateInProgressChannel
			now := time.Now()
			executed <- now
			time.Sleep(updateDuration)
			res <- updateResponse{now.UnixNano(), nil}
		}
	}()

	update := func(index int, expectError bool) {
		go func() {
			_, err := repo.tryUpdate()
			assert.Equal(t, expectError, err != nil, "Error in request %d", index)
		}()
	}

	// First one is being executed
	update(0, false)
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, 1, len(executed))

	// Second one is being queued
	update(1, false)
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, 1, len(executed))

	// Last one gets "Accepted"
	time.Sleep(1 * time.Millisecond)
	lastUpdateRequest := time.Now()
	update(2, true)
	assert.Equal(t, 1, len(executed))

	<-executed // Dump first execution time

	// Second execution should be done after given sleep time
	time.Sleep(updateDuration)
	assert.Equal(t, 1, len(executed))
	lastUpdate := <-executed
	assert.True(t, lastUpdate.After(lastUpdateRequest))

	// When all requests are complete, allow queueing up again
	update(3, false)
}
