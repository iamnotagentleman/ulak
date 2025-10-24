package middleware

import (
	"net/http"
	"sync"
)

type requestSerializer struct {
	mutexes map[string]*sync.Mutex
	mapMu   sync.RWMutex
}

// newRequestSerializer creates a new request serializer
func newRequestSerializer() *requestSerializer {
	return &requestSerializer{
		mutexes: make(map[string]*sync.Mutex),
	}
}

// getMutex gets or creates a mutex for the given key
func (rs *requestSerializer) getMutex(key string) *sync.Mutex {
	rs.mapMu.RLock()
	mu, exists := rs.mutexes[key]
	rs.mapMu.RUnlock()

	if exists {
		return mu
	}

	rs.mapMu.Lock()
	defer rs.mapMu.Unlock()

	if mu, exists := rs.mutexes[key]; exists {
		return mu
	}

	// Create new mutex
	mu = &sync.Mutex{}
	rs.mutexes[key] = mu
	return mu
}

var globalSerializer = newRequestSerializer()

// SerializeRequestsMiddleware ensures only one request at a time can execute for a specific route
// This prevents race conditions
func SerializeRequestsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// route-specific locking
		routeKey := r.URL.Path

		mu := globalSerializer.getMutex(routeKey)
		mu.Lock()
		defer mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
