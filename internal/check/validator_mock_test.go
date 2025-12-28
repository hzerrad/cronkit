package check

import (
	"time"

	"github.com/hzerrad/cronic/internal/crontab"
)

// mockReader is a mock implementation of crontab.Reader for testing
type mockReader struct {
	jobs    []*crontab.Job
	entries []*crontab.Entry
	err     error
}

func (m *mockReader) ReadFile(path string) ([]*crontab.Job, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.jobs, nil
}

func (m *mockReader) ReadUser() ([]*crontab.Job, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.jobs, nil
}

func (m *mockReader) ParseFile(path string) ([]*crontab.Entry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.entries, nil
}

// mockScheduler is a mock implementation of cronx.Scheduler for testing empty schedules
type mockScheduler struct {
	returnEmpty bool
	returnError bool
}

func (m *mockScheduler) Next(expression string, from time.Time, count int) ([]time.Time, error) {
	if m.returnError {
		return nil, &mockError{msg: "mock error"}
	}
	if m.returnEmpty {
		// Return a time far in the future to simulate empty schedule
		return []time.Time{from.AddDate(3, 0, 0)}, nil
	}
	// Return a normal time
	return []time.Time{from.Add(time.Hour)}, nil
}

// mockError is a simple error type for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
