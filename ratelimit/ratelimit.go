package ratelimit

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type RateLimitedClient struct {
	client  *http.Client
	lock    sync.RWMutex
	buckets map[string]*time.Timer
}

func New(client *http.Client) *RateLimitedClient {
	return &RateLimitedClient{
		buckets: make(map[string]*time.Timer),
		client:  client,
	}
}

func (r *RateLimitedClient) doNow(key string, request *http.Request) (*http.Response, error) {
	response, err := r.client.Do(request)
	if response != nil {
		remaining := response.Header.Get("X-Ratelimit-Remaining")
		reset := response.Header.Get("X-Ratelimit-Reset")

		now := time.Now()
		remainingInt, err := strconv.Atoi(remaining)
		if err != nil {
			return nil, err
		}
		resetInt, err := strconv.Atoi(reset)
		if err != nil {
			return nil, err
		}
		resetTime := time.Unix(int64(resetInt), 0)
		duration := (resetTime.Sub(now) - (5 * time.Second)) / time.Duration(remainingInt)
		log.Printf("Time resets at: %s. Waiting for %s.\n", resetTime, duration)

		r.lock.Lock()
		t := r.buckets[key]
		if t == nil {
			r.buckets[key] = time.NewTimer(duration)
		} else {
			t.Reset(duration)
		}
		r.lock.Unlock()
	}

	return response, err
}

func (r *RateLimitedClient) Do(key string, request *http.Request) (*http.Response, error) {
	return r.doNow(key, request)
}

func (r *RateLimitedClient) DoWait(key string, request *http.Request) (*http.Response, error) {
	r.lock.RLock()
	t := r.buckets[key]
	if t != nil {
		<-t.C
	}
	r.lock.RUnlock()

	return r.doNow(key, request)
}
