package cache

import (
	"sync"
)

// Call represents a single in-flight or completed call
type Call struct {
	wg   sync.WaitGroup
	val  interface{}
	err  error
	dups int
}

// SingleFlight provides request coalescing for concurrent identical requests
// This prevents the "thundering herd" problem where multiple goroutines
// request the same resource simultaneously
type SingleFlight struct {
	mu sync.Mutex
	m  map[string]*Call
}

// NewSingleFlight creates a new SingleFlight instance
func NewSingleFlight() *SingleFlight {
	return &SingleFlight{
		m: make(map[string]*Call),
	}
}

// Do executes and returns the results of the given function, making sure
// that only one execution is in-flight for a given key at a time.
// If a duplicate comes in, the duplicate caller waits for the original
// to complete and receives the same results.
func (sf *SingleFlight) Do(key string, fn func() (interface{}, error)) (interface{}, error, bool) {
	sf.mu.Lock()
	if sf.m == nil {
		sf.m = make(map[string]*Call)
	}

	if c, ok := sf.m[key]; ok {
		c.dups++
		sf.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err, true // shared is true
	}

	c := &Call{}
	c.wg.Add(1)
	sf.m[key] = c
	sf.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	sf.mu.Lock()
	delete(sf.m, key)
	sf.mu.Unlock()

	return c.val, c.err, false // shared is false
}

// DoAsync is like Do but doesn't wait for the result if another
// call is in-flight. Returns immediately if a duplicate request.
func (sf *SingleFlight) DoAsync(key string, fn func() (interface{}, error)) <-chan Result {
	ch := make(chan Result, 1)

	go func() {
		val, err, _ := sf.Do(key, fn)
		ch <- Result{Val: val, Err: err}
	}()

	return ch
}

// Forget removes a key from the in-flight map
func (sf *SingleFlight) Forget(key string) {
	sf.mu.Lock()
	delete(sf.m, key)
	sf.mu.Unlock()
}

// Result holds the result of a singleflight call
type Result struct {
	Val interface{}
	Err error
}

// InFlight returns the number of in-flight calls
func (sf *SingleFlight) InFlight() int {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	return len(sf.m)
}
