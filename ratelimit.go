package csfloat

import (
	"net/http"
	"strconv"
	"time"
)

type Ratelimits struct {
	Limit         uint
	Remaining     uint
	Reset         time.Time
	SuggestedWait time.Time
}

func RatelimitsFrom(response *http.Response) (Ratelimits, error) {
	var ratelimits Ratelimits

	remaining := response.Header.Get("X-Ratelimit-Remaining")
	limit := response.Header.Get("X-Ratelimit-Limit")
	reset := response.Header.Get("X-Ratelimit-Reset")

	now := time.Now()
	remainingInt, err := strconv.ParseUint(remaining, 10, 64)
	if err != nil {
		return ratelimits, err
	}

	limitInt, err := strconv.ParseUint(limit, 10, 64)
	if err != nil {
		return ratelimits, err
	}

	resetInt, err := strconv.Atoi(reset)
	if err != nil {
		return ratelimits, err
	}

	ratelimits.Remaining = uint(remainingInt)
	ratelimits.Limit = uint(limitInt)
	resetTime := time.Unix(int64(resetInt), 0)
	ratelimits.Reset = resetTime
	ratelimits.SuggestedWait = now.Add(resetTime.Sub(now) / time.Duration(remainingInt))

	return ratelimits, nil
}
