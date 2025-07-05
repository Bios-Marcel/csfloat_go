package csfloat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Fee is a constant fee. Technically the profile has a setting, but it seems
// its unachievable to reduce the fee, so this is fine for now.
const Fee float64 = 2

type CSFloat struct {
	httpClient *http.Client
}

func New() *CSFloat {
	client := http.Client{}
	return &CSFloat{
		httpClient: &client,
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

type InventoryItem struct {
	Item
	Reference Reference `json:"reference"`
}

type Sticker struct {
	Index uint   `json:"stickerId"`
	Name  string `json:"name"`
}

type Charm struct {
	// CharmId is called stickerId, not a typo.
	Index   uint   `json:"stickerId"`
	Pattern uint   `json:"pattern"`
	Name    string `json:"name"`
}

type ItemType string

const (
	TypeCharm ItemType = "charm"
	TypeSkin  ItemType = "skin"
)

type Item struct {
	ID             string   `json:"asset_id"`
	Rarity         Rarity   `json:"rarity"`
	Type           ItemType `jsob:"type"`
	MarketHashName string   `json:"market_hash_name"`

	Float      float64   `json:"float_value"`
	IsSouvenir bool      `json:"is_souvenir"`
	PaintIndex uint      `json:"paint_index"`
	Stickers   []Sticker `json:"stickers"`
	Charms     []Charm   `json:"keychains"`

	CharmIndex   uint `json:"keychain_index"`
	CharmPattern uint `json:"keychain_pattern"`
}

type ListingsResponse struct {
	Ratelimits Ratelimits
	Data       []ListedItem `json:"data"`
}

type ListingsRequest struct {
	MinPrice uint
	MaxPrice uint
	MaxFloat float32
}

func (api *CSFloat) Listings(apiKey string, query ListingsRequest) (*ListingsResponse, error) {
	endpoint := "https://csfloat.com/api/v1/listings"
	request, err := http.NewRequest(
		http.MethodGet,
		endpoint,
		nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	form := url.Values{}
	form.Set("type", "buy_now")
	form.Set("sort_by", "highest_discount")
	form.Set("limit", "40")
	form.Set("min_price", fmt.Sprintf("%d", query.MinPrice))
	form.Set("max_price", fmt.Sprintf("%d", query.MaxPrice))
	form.Set("max_float", fmt.Sprintf("%f", query.MaxFloat))
	request.URL.RawQuery = form.Encode()

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	var result ListingsResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	ratelimits, err := RatelimitsFrom(response)
	if err != nil {
		return nil, fmt.Errorf("error getting ratelimits: %w", err)
	}
	result.Ratelimits = ratelimits

	return &result, nil
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

	response, err := api.httpClient.Do(request)
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

func (api *CSFloat) Inventory(apiKey string) ([]InventoryItem, error) {
	endpoint := "https://csfloat.com/api/v1/me/inventory"
	request, err := http.NewRequest(
		http.MethodGet,
		endpoint,
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

	var items []InventoryItem
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
	endpoint := "https://csfloat.com/api/v1/listings"
	request, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/%s", endpoint, listingId),
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

	response, err := api.httpClient.Do(request)
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

// List puts an item up for sale
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

	response, err := api.httpClient.Do(request)
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

type TradeState string

const (
	// Verified means both sides have received the goods.
	Verified TradeState = "verified"
	// Cancelled means the buyer decided not to buy afterall.
	Cancelled TradeState = "cancelled"
	// Failed means the buyer failed to accept
	Failed TradeState = "failed"
)

type Trade struct {
	ID string `json:"id"`
	// BuyerId is the steam ID, which can be your own ID if you are the buyer.
	BuyerId    string     `json:"buyer_id"`
	Contract   ListedItem `json:"contract"`
	VerifiedAt time.Time  `json:"verified_at"`
	State      TradeState `json:"state"`
}

type TradesResponse struct {
	Ratelimits Ratelimits
	Trades     []Trade `json:"trades"`
	Count      uint    `json:"count"`
}

type TradesRequest struct {
	// Page, default 0 (latest)
	Page uint `json:"page"`
	// Limit, default 100
	Limit uint `json:"limit"`
	// States, empty by default, not filtering
	States []TradeState
}

func (api *CSFloat) Trades(apiKey string, payload TradesRequest) (*TradesResponse, error) {
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

	form := url.Values{}
	if len(payload.States) > 0 {
		var str strings.Builder
		for index, state := range payload.States {
			str.WriteString(string(state))
			if index < len(payload.States)-1 {
				str.WriteRune(',')
			}
		}
		form.Set("state", str.String())
	}
	form.Set("page", strconv.FormatUint(uint64(payload.Page), 10))
	form.Set("limit", strconv.FormatUint(uint64(payload.Limit), 10))
	request.URL.RawQuery = form.Encode()

	fmt.Println(request.URL.RawQuery)

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid return code: %d", response.StatusCode)
	}

	var result TradesResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	ratelimits, err := RatelimitsFrom(response)
	if err != nil {
		return nil, fmt.Errorf("error getting ratelimits")
	}

	result.Ratelimits = ratelimits

	return &result, nil
}

type HistoryEntry struct {
	Pruce  uint      `json:"price"`
	Item   Item      `json:"item"`
	SoldAt time.Time `json:"sold_at"`
}

type HistoryRequestPayload struct {
	MarketHashName string
	PaintIndex     uint
}

type HistoryResponse struct {
	Ratelimits Ratelimits
	Data       []HistoryEntry
}

func (api *CSFloat) History(apiKey string, payload HistoryRequestPayload) (*HistoryResponse, error) {
	endpoint := "https://csfloat.com/api/v1/history"
	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/%s/sales", endpoint, url.QueryEscape(payload.MarketHashName)),
		nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	form := url.Values{}
	form.Set("paint_index", strconv.FormatUint(uint64(payload.PaintIndex), 10))
	request.URL.RawQuery = form.Encode()

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid return code: %d", response.StatusCode)
	}

	var result []HistoryEntry
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	ratelimits, err := RatelimitsFrom(response)
	if err != nil {
		return nil, fmt.Errorf("error getting ratelimits: %w", err)
	}

	return &HistoryResponse{
		Ratelimits: ratelimits,
		Data:       result,
	}, nil
}
