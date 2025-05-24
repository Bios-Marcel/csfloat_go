package csfloat_test

import (
	"os"
	"sync"
	"testing"

	csfloat "github.com/Bios-Marcel/csfloat_go"
	"github.com/stretchr/testify/assert"
)

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
	t.Log(items)
	if ass.NoError(err) {
		ass.NotEmpty(items)
	}
}

func Test_Stall(t *testing.T) {
	ass := assert.New(t)
	api := csfloat.New()

	items, err := api.Stall(apiKey, me().User.SteamId)
	t.Log(items)
	if ass.NoError(err) {
		ass.NotEmpty(items)
	}
}
