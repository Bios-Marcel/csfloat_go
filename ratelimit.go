package csfloat

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// RatelimitBucketKey identifies a ratelimit bucket. Each API endpoint gets its
// own key so that per-endpoint and global ratelimits can be tracked separately.
type RatelimitBucketKey string

const (
	RatelimitKeyGetListing             RatelimitBucketKey = "get_listing"
	RatelimitKeyGetListings            RatelimitBucketKey = "get_listings"
	RatelimitKeyGetStall               RatelimitBucketKey = "get_stall"
	RatelimitKeyGetInventory           RatelimitBucketKey = "get_inventory"
	RatelimitKeyGetMe                  RatelimitBucketKey = "get_me"
	RatelimitKeyPostNewOffer           RatelimitBucketKey = "post_new_offer"
	RatelimitKeyBulkAcceptTrade        RatelimitBucketKey = "bulk_accept_trade"
	RatelimitKeyBulkCancel             RatelimitBucketKey = "bulk_cancel"
	RatelimitKeyBulkList               RatelimitBucketKey = "bulk_list"
	RatelimitKeyBulkUnlist             RatelimitBucketKey = "bulk_unlist"
	RatelimitKeyUnlist                 RatelimitBucketKey = "unlist"
	RatelimitKeyUpdateListing          RatelimitBucketKey = "update_listing"
	RatelimitKeyGetTrades              RatelimitBucketKey = "get_trades"
	RatelimitKeyGetHistory             RatelimitBucketKey = "get_history"
	RatelimitKeyBuy                    RatelimitBucketKey = "buy"
	RatelimitKeyUnwatch                RatelimitBucketKey = "unwatch"
	RatelimitKeyWatch                  RatelimitBucketKey = "watch"
	RatelimitKeyGetItemBuyOrders       RatelimitBucketKey = "get_item_buy_orders"
	RatelimitKeyGetSimpleItemBuyOrders RatelimitBucketKey = "get_simple_item_buy_orders"
	RatelimitKeyGetListingBuyOrders    RatelimitBucketKey = "get_listing_buy_orders"
	RatelimitKeyGetSimilar             RatelimitBucketKey = "get_similar"
	RatelimitKeyGetTransactions        RatelimitBucketKey = "get_transactions"
	RatelimitKeyCreateListing          RatelimitBucketKey = "create_listing"
	RatelimitKeyCreateBuyOrder         RatelimitBucketKey = "create_buy_order"
	RatelimitKeyDeleteBuyOrder         RatelimitBucketKey = "delete_buy_order"
)

type Ratelimits struct {
	Limit         uint
	Remaining     uint
	Reset         time.Time
	SuggestedWait time.Time
}

// BucketRatelimits returns the most recent ratelimit data for the given bucket,
// or nil if no request has been made for that bucket yet.
func (api *API) BucketRatelimits(key RatelimitBucketKey) *Ratelimits {
	return api.ratelimits[key]
}

// IsGloballyRatelimited reports whether we appear to be hitting a global (cross-
// endpoint) ratelimit. It detects this by checking if any bucket's remaining
// count dropped from to 0 and the reset time is in the future and at most 5
// minutes away (the known global reset window).
func (api *API) IsGloballyRatelimited() bool {
	now := time.Now()
	for _, entry := range api.ratelimits {
		if entry != nil && entry.Remaining == 0 {
			resetIn := entry.Reset.Sub(now)
			if resetIn > 0 && resetIn <= 5*time.Minute {
				return true
			}
		}
	}
	return false
}

func (api *API) updateRatelimits(key RatelimitBucketKey, ratelimits *Ratelimits) {
	api.ratelimits[key] = ratelimits
}

// ratelimitsFrom parses the headers to get ratelimiting. It does NOT consume
// the body.
func ratelimitsFrom(response *http.Response) (Ratelimits, error) {
	var ratelimits Ratelimits

	remaining := response.Header.Get("X-Ratelimit-Remaining")
	limit := response.Header.Get("X-Ratelimit-Limit")
	reset := response.Header.Get("X-Ratelimit-Reset")

	now := time.Now()
	remainingInt, err := strconv.ParseUint(remaining, 10, 64)
	if err != nil {
		return ratelimits, fmt.Errorf("error parsing X-Ratelimit-Remaining: %w", err)
	}

	limitInt, err := strconv.ParseUint(limit, 10, 64)
	if err != nil {
		return ratelimits, fmt.Errorf("error parsing X-Ratelimit-Limit: %w", err)
	}

	resetInt, err := strconv.Atoi(reset)
	if err != nil {
		return ratelimits, fmt.Errorf("error parsing X-Ratelimit-Reset: %w", err)
	}

	ratelimits.Remaining = uint(remainingInt)
	ratelimits.Limit = uint(limitInt)
	resetTime := time.Unix(int64(resetInt), 0)
	ratelimits.Reset = resetTime
	if remainingInt == 0 {
		ratelimits.SuggestedWait = resetTime
	} else {
		ratelimits.SuggestedWait = now.Add(resetTime.Sub(now) / time.Duration(remainingInt))
	}

	return ratelimits, nil
}
