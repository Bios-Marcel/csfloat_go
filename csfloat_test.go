package csfloat_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	csfloat "github.com/Bios-Marcel/csfloat_go"
	"github.com/stretchr/testify/assert"
)

func TestDummy(t *testing.T) {
	t.Skip()

	dialer := &net.Dialer{
		Timeout: 15 * time.Second,
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, "tcp4", addr)
			if err == nil {
				log.Println(conn.RemoteAddr().String())
				return conn, nil
			}

			return nil, fmt.Errorf("no IPv6 address could be dialed for %s", addr)
		},
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 3 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	request, err := http.NewRequest("GET", "https://csfloat.com", nil)
	if err != nil {
		panic(err)
	}
	client.Do(request)

}

func TestJsonEncoder(t *testing.T) {
	b := csfloat.ListRequest{
		BuyNowRequest: &csfloat.BuyNowRequest{
			Price: 5,
		},
	}
	x, _ := json.MarshalIndent(b, "", "  ")
	t.Log(string(x))
}

var apiKey = os.Getenv("CSFLOAT_API_KEY")
var me = sync.OnceValue(func() *csfloat.MeResponse {
	api := csfloat.New()
	me, err := api.Me(apiKey)
	if err != nil {
		panic(err)
	}
	return me
})

func Test_Inventory(t *testing.T) {
	ass := assert.New(t)
	api := csfloat.New()

	items, err := api.Inventory(apiKey)
	if ass.NoError(err) {
		t.Log(items)
		ass.NotEmpty(items)
	}
}

func Test_Stall(t *testing.T) {
	ass := assert.New(t)
	api := csfloat.New()

	items, err := api.Stall(apiKey, me().User.SteamId)
	if ass.NoError(err) {
		t.Log(items)
		ass.NotEmpty(items)
	}
}

func Test_Listing(t *testing.T) {
	ass := assert.New(t)
	api := csfloat.New()

	listing, err := api.Listing(apiKey, "869907646323492200")
	if ass.NoError(err) {
		t.Log(listing.Item)
		ass.NotEmpty(listing)
	}
}

func Test_Listings(t *testing.T) {
	ass := assert.New(t)
	api := csfloat.New()

	items, err := api.Listings(apiKey, csfloat.ListingsRequest{})
	if ass.NoError(err) {
		t.Log(items)
		ass.NotEmpty(items)
	}
}

func Test_BuyIncorrectPrice(t *testing.T) {
	// ass := assert.New(t)
	api := csfloat.New()

	response, err := api.Buy(apiKey, csfloat.BuyRequestPayload{
		ContractIds: []string{"861537163265837281"},
		TotalPrice:  1,
	})
	t.Log(response.Error, err)
}
