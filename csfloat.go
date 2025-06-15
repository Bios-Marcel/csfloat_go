package csfloat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Bios-Marcel/csfloat_go/ratelimit"
)

// FIXME Use fee from /me?
const Fee float64 = 2

type CSFloat struct {
	httpClient *ratelimit.RateLimitedClient
}

func New() *CSFloat {
	client := http.Client{}
	// FIXME Set timeout params?
	return &CSFloat{
		httpClient: ratelimit.New(&client),
	}
}

type Stall struct {
	Items []ListedItem `json:"data"`
	Count int          `json:"total_count"`
}

type AuctionType string

const (
	BuyNow  AuctionType = "buy_now"
	Auction AuctionType = "auction"
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

type Reference struct {
	BasePrice      uint `json:"base_price"`
	PredictedPrice uint `json:"predicted_price"`
	Quantity       uint `json:"quantity"`
}

type ListedItem struct {
	ID               string      `json:"id"`
	Price            uint        `json:"price"`
	Item             Item        `json:"item"`
	Reference        Reference   `json:"reference"`
	Type             AuctionType `json:"type"`
	Description      string      `json:"description"`
	Private          bool        `json:"private"`
	MaxOfferDiscount uint        `json:"max_offer_discount"`
}

type Item struct {
	ID         string  `json:"asset_id"`
	Float      float64 `json:"float_value"`
	Rarity     Rarity  `json:"rarity"`
	IsSouvenir bool    `json:"is_souvenir"`
}

type listingsResponse struct {
	Data []ListedItem `json:"data"`
}
type ListingsRequest struct {
	MinPrice uint
	MaxPrice uint
	MaxFloat float32
}

func (api *CSFloat) Listings(apiKey string, query ListingsRequest) ([]ListedItem, error) {
	endpoint := "https://csfloat.com/api/v1/listings"
	request, err := http.NewRequest(
		http.MethodGet,
		endpoint,
		nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	request.Form = url.Values{}
	request.Form.Set("type", "buy_now")
	request.Form.Set("sort_by", "highest_discount")
	request.Form.Set("limit", "40")
	request.Form.Set("min_price", fmt.Sprintf("%d", query.MinPrice))
	request.Form.Set("max_price", fmt.Sprintf("%d", query.MaxPrice))
	request.Form.Set("max_float", fmt.Sprintf("%f", query.MaxFloat))

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.DoWait(endpoint+apiKey, request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	var result listingsResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	return result.Data, nil
}

func (api *CSFloat) Stall(apiKey, steamId string) (*Stall, error) {
	endpoint := fmt.Sprintf("https://csfloat.com/api/v1/users/%s/stall", steamId)
	request, err := http.NewRequest(
		http.MethodGet,
		endpoint+"?limit=40",
		nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(endpoint+apiKey, request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	if response.StatusCode > 200 {
		return nil, fmt.Errorf("bad response: %d: %s", response.StatusCode, mustString(response))
	}

	var stall Stall
	if err := json.NewDecoder(response.Body).Decode(&stall); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &stall, nil
}

func mustString(response *http.Response) string {
	b, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (api *CSFloat) Inventory(apiKey string) ([]Item, error) {
	endpoint := "https://csfloat.com/api/v1/me/inventory"
	request, err := http.NewRequest(
		http.MethodGet,
		endpoint,
		nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(endpoint+apiKey, request)
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
	endpoint := "https://csfloat.com/api/v1/me"
	request, err := http.NewRequest(
		http.MethodGet,
		endpoint,
		nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(endpoint+apiKey, request)
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
	endpoint := "https://csfloat.com/api/v1/listings"
	request, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/%s", endpoint, listingId),
		nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(endpoint+apiKey, request)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}

	// FIXME Get body?
	if response.StatusCode != 200 {
		return fmt.Errorf("invalid status code: %d", response.StatusCode)
	}

	return nil
}

type UpdateListingRequest struct {
	MaxOfferDiscount uint `json:"max_offer_discount"`
}

func (api *CSFloat) UpdateListing(apiKey, id string, payload UpdateListingRequest) error {
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(payload); err != nil {
		return fmt.Errorf("error encoding payload: %w", err)
	}

	endpoint := "https://csfloat.com/api/v1/listings"
	request, err := http.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("%s/%s", endpoint, id),
		&buffer)

	request.Header.Set("Authorization", apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Length", strconv.Itoa(buffer.Len()))

	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	response, err := api.httpClient.Do(endpoint+apiKey, request)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("invalid status code: %d", response.StatusCode)
	}

	return nil
}

type BuyNowRequest struct {
	Price uint `json:"price,omitempty"`
}

type AuctionRequest struct {
	DurationDays uint `json:"duration_days,omitempty"`
	ReservePrice uint `json:"reserve_price,omitempty"`
}

type ListRequest struct {
	*BuyNowRequest  `json:",omitempty"`
	*AuctionRequest `json:",omitempty"`

	AssetId     string      `json:"asset_id"`
	AuctionType AuctionType `json:"type"`
	Description string      `json:"description"`
}

func (api *CSFloat) List(apiKey string, payload ListRequest) (*ListedItem, error) {
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(payload); err != nil {
		return nil, fmt.Errorf("error encoding payload: %w", err)
	}

	endpoint := "https://csfloat.com/api/v1/listings"
	request, err := http.NewRequest(
		http.MethodPost,
		endpoint,
		&buffer)

	request.Header.Set("Authorization", apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Length", strconv.Itoa(buffer.Len()))

	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	response, err := api.httpClient.Do(endpoint+apiKey, request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	// FIXME Get body?
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("invalid status code: %d", response.StatusCode)
	}

	var result ListedItem
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response, item was relisted: %w", err)
	}

	return &result, nil
}

type Trade struct {
	// BuyerId is the steam ID, which can be your own ID if you are the buyer.
	BuyerId  string     `json:"buyer_id"`
	Contract ListedItem `json:"item"`
}

type TradesResponse struct {
	Trades []Trade `json:"trades"`
	Count  uint    `json:"count"`
}

type TradesRequest struct {
	// Page, default 0 (latest)
	Page uint `json:"page"`
	// Limit, default 100
	Limit uint `json:"limit"`
}

func (api *CSFloat) Trades(apiKey string, payload TradesRequest) (*ListedItem, error) {
	endpoint := "https://csfloat.com/api/v1/me/trades"
	request, err := http.NewRequest(
		http.MethodGet,
		endpoint,
		nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	if payload.Limit == 0 {
		payload.Limit = 100
	}

	request.Form = url.Values{}
	request.Form.Set("page", strconv.FormatUint(10, int(payload.Page)))
	request.Form.Set("limit", strconv.FormatUint(10, int(payload.Limit)))

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.DoWait(endpoint+apiKey, request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	var result listingsResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	return result.Data, nil
}
