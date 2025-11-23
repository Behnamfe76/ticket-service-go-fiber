package observability

import (
	"strconv"
	"sync"
	"time"
)

// Metrics provides basic in-memory counters.
type Metrics struct {
	mu           sync.Mutex
	requestCount map[string]int64
	errorCount   map[string]int64
}

// NewMetrics initializes metrics storage.
func NewMetrics() *Metrics {
	return &Metrics{
		requestCount: make(map[string]int64),
		errorCount:   make(map[string]int64),
	}
}

// RecordRequest increments counters for requests.
func (m *Metrics) RecordRequest(path, method string, status int, duration time.Duration) {
	if m == nil {
		return
	}
	key := pathKey(path, method, status)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestCount[key]++
}

// RecordError increments error counters.
func (m *Metrics) RecordError(path, method, code string) {
	if m == nil {
		return
	}
	key := path + "|" + method + "|" + code
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCount[key]++
}

func pathKey(path, method string, status int) string {
	return path + "|" + method + "|" + strconv.Itoa(status)
}
