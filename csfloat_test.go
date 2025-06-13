package csfloat_test

import (
	"encoding/json"
	"os"
	"sync"
	"testing"

	csfloat "github.com/Bios-Marcel/csfloat_go"
	"github.com/stretchr/testify/assert"
)

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
var me = sync.OnceValue(func() *csfloat.Me {
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

func Test_Listings(t *testing.T) {
	ass := assert.New(t)
	api := csfloat.New()

	items, err := api.Listings(apiKey, csfloat.ListingsRequest{})
	if ass.NoError(err) {
		t.Log(items)
		ass.NotEmpty(items)
	}
}
