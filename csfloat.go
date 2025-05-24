package csfloat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// FIXME Use fee from /me?
const Fee float64 = 2

type CSFloat struct {
	httpClient *http.Client
}

func New() *CSFloat {
	client := http.Client{}
	// FIXME Set timeout params?
	return &CSFloat{
		httpClient: &client,
	}
}

type Stall struct {
	Items []StallItem `json:"data"`
	Count int         `json:"total_count"`
}

type AuctionType string

const (
	BuyNow AuctionType = "buy_now"
)

type Rarity uint8

const (
	Consumer   Rarity = 0
	Industrial Rarity = 1
	MilSpec    Rarity = 2
	Restricted Rarity = 3
	Classified Rarity = 4
	Covert     Rarity = 5
	Contraband Rarity = 6
)

type WearName string

const (
	BattleScarred WearName = "Battle-Scarred"
	WellWorn      WearName = "Well-Worn"
	FieldTested   WearName = "Field-Tested"
	MinimalWear   WearName = "Minimal Wear"
	FactoryNew    WearName = "Factory New"
)

type Item struct {
	ID     string  `json:"asset_id"`
	Float  float64 `json:"float_value"`
	Rarity Rarity  `json:"rarity"`

	// FIXME There's no int value, just strings, which are potentially
	// translated. BUT, we can just set a hardcoded language in our requests.
	WearName  WearName `json:"wear_name"`
	ListingId string   `json:"listing_id"`
}

type StallItem struct {
	ID          string      `json:"id"`
	Watchers    uint        `json:"watchers"`
	Price       uint        `json:"price"`
	AuctionType AuctionType `json:"type"`
	CreatedAt   time.Time   `json:"created_at"`
	Item        Item        `json:"item"`
	// FIXME Reference
}

func (api *CSFloat) Stall(apiKey, steamId string) (*Stall, error) {
	request, err := http.NewRequest(
		http.MethodGet,
		// FIXME Adjustable limit
		// FIXME Paging?
		fmt.Sprintf("https://csfloat.com/api/v1/users/%s/stall?limit=40", steamId),
		nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	var stall Stall
	if err := json.NewDecoder(response.Body).Decode(&stall); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &stall, nil
}

func (api *CSFloat) Inventory(apiKey string) ([]Item, error) {
	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://csfloat.com/api/v1/me/inventory"),
		nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	// It returns everything, including what's in stall.
	// tradeable will always be 1, since steam does not shot untradable items
	// anymore. Items with a `listing_id` are already in your stall.

	var items []Item
	if err := json.NewDecoder(response.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return items, nil
}

type MeUser struct {
	SteamId        string `json:"steam_id"`
	Balance        uint   `json:"balance"`
	PendingBalance uint   `json:"pending_balance"`
}

type Me struct {
	User MeUser `json:"user"`
}

func (api *CSFloat) Me(apiKey string) (*Me, error) {
	request, err := http.NewRequest(
		http.MethodGet,
		"https://csfloat.com/api/v1/me",
		nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	var me Me
	if err := json.NewDecoder(response.Body).Decode(&me); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	return &me, nil
}

func (api *CSFloat) Unlist(apiKey, listingId string) error {
	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://csfloat.com/api/v1/listings/%s", listingId),
		nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}

	// FIXME Get body?
	if response.StatusCode != 200 {
		return fmt.Errorf("invalid status code: %d", response.StatusCode)
	}

	return nil
}

type ListRequest struct {
	AssetId     string      `json:"asset_id"`
	Price       uint        `json:"price"`
	AuctionType AuctionType `json:"type"`
}

func (api *CSFloat) List(apiKey string, payload ListRequest) error {
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(payload); err != nil {
		return fmt.Errorf("error encoding payload: %w", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		"https://csfloat.com/api/v1/listings",
		&buffer)

	request.Header.Set("Authorization", apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Length", strconv.Itoa(buffer.Len()))

	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	response, err := api.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}

	// FIXME Get body?
	if response.StatusCode != 200 {
		return fmt.Errorf("invalid status code: %d", response.StatusCode)
	}

	return nil
}
