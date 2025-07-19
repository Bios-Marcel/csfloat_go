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

type ListingType string

const (
	BuyNow  ListingType = "buy_now"
	Auction ListingType = "auction"
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

type ItemReference struct {
	BasePrice      uint `json:"base_price"`
	PredictedPrice uint `json:"predicted_price"`
	Quantity       uint `json:"quantity"`
}

type Reference struct {
	Price    uint `json:"price"`
	Quantity uint `json:"quantity"`
}

type ListedItem struct {
	ID               string        `json:"id"`
	Price            uint          `json:"price"`
	Item             Item          `json:"item"`
	Reference        ItemReference `json:"reference"`
	Type             ListingType   `json:"type"`
	Description      string        `json:"description"`
	Private          bool          `json:"private"`
	MaxOfferDiscount uint          `json:"max_offer_discount"`
	Watchers         uint          `json:"watchers"`
}

type InventoryItem struct {
	Item
	Reference ItemReference `json:"reference"`
}

type Sticker struct {
	Index     uint      `json:"stickerId"`
	Name      string    `json:"name"`
	Reference Reference `json:"reference"`
	ItemURL   string    `json:"item_url"`
	Wear      float32   `json:"wear"`
	Rotation  float32   `json:"rotation"`
}

type Charm struct {
	// CharmId is called stickerId, not a typo.
	Index     uint      `json:"stickerId"`
	Pattern   uint      `json:"pattern"`
	Name      string    `json:"name"`
	Reference Reference `json:"reference"`
}

type ItemType string

const (
	TypeCharm ItemType = "charm"
	TypeSkin  ItemType = "skin"
)

type Fade struct {
	// Seed sems to be the same as the paintseed so far.
	Seed       uint    `json:"seed"`
	Percentage float64 `json:"percentage"`
	Rank       uint    `json:"rank"`
	Type       string  `json:"type"`
}

type BlueGem struct {
	PlaysideBlue   float64 `json:"playside_blue"`
	PlaysideGold   float64 `json:"playside_gold"`
	PlaysidePurple float64 `json:"playside_purple"`
	BacksideBlue   float64 `json:"backside_blue"`
	BacksideGold   float64 `json:"backside_gold"`
	BacksidePurple float64 `json:"backside_purple"`
}

type Item struct {
	ID             string   `json:"asset_id"`
	Rarity         Rarity   `json:"rarity"`
	Type           ItemType `jsob:"type"`
	MarketHashName string   `json:"market_hash_name"`

	Float      float64 `json:"float_value"`
	IsStattrak bool    `json:"is_stattrak"`
	IsSouvenir bool    `json:"is_souvenir"`
	// DefIndex is the weapon type
	DefIndex uint `json:"def_index"`
	// PaintIndex is the skin type
	PaintIndex uint `json:"paint_index"`
	// PaintSeed determines the skin pattern
	PaintSeed uint      `json:"paint_seed"`
	Stickers  []Sticker `json:"stickers"`
	Charms    []Charm   `json:"keychains"`
	Fade      *Fade     `json:"fade"`
	BlueGem   *BlueGem  `json:"blue_gem"`

	CharmIndex   uint `json:"keychain_index"`
	CharmPattern uint `json:"keychain_pattern"`
}

// Category will map to the query category matching this item. This is required
// for listing similar items.
func (item *Item) Category() Category {
	if item.IsSouvenir {
		return Souvenir
	}
	if item.IsStattrak {
		return StatTrak
	}
	return Normal
}

type ListingsResponse struct {
	Ratelimits Ratelimits
	Data       []ListedItem `json:"data"`
}

type Category uint

const (
	Normal   = 1
	StatTrak = 2
	Souvenir = 3
)

type ListingsRequest struct {
	MinPrice    uint
	MaxPrice    uint
	MinFloat    float32
	MaxFloat    float32
	ExcludeRare bool
	Category    Category
	DefIndex    uint
	PaintIndex  uint
}

// FloatRange returns the float range for the given quality (fn, mw, ...).
func (api *CSFloat) FloatRange(f float32) (float32, float32) {
	if f < 0.07 {
		return 0.0, 0.07
	} else if f < 0.15 {
		return 0.07, 0.15
	} else if f < 0.38 {
		return 0.15, 0.38
	} else if f < 0.45 {
		return 0.38, 0.45
	}

	return 0.45, 1.0
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
	if query.ExcludeRare {
		form.Set("min_ref_qty", strconv.FormatUint(20, 10))
	}
	if query.DefIndex > 0 {
		form.Set("def_index", strconv.FormatUint(uint64(query.DefIndex), 10))
	}
	if query.PaintIndex > 0 {
		form.Set("paint_index", strconv.FormatUint(uint64(query.PaintIndex), 10))
	}
	if query.Category > 0 {
		form.Set("category", strconv.FormatUint(uint64(query.Category), 10))
	}
	if query.MinPrice > 0 {
		form.Set("min_price", fmt.Sprintf("%d", query.MinPrice))
	}
	if query.MaxPrice > 0 {
		form.Set("max_price", fmt.Sprintf("%d", query.MaxPrice))
	}
	form.Set("min_float", fmt.Sprintf("%0f", query.MinFloat))
	// FIXME Set max float to 0.99 to avoid charms and stickers?
	if query.MaxFloat > 0 {
		form.Set("max_float", fmt.Sprintf("%0f", query.MaxFloat))
	}
	request.URL.RawQuery = form.Encode()

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	var result ListingsResponse
	ratelimits, err := RatelimitsFrom(response)
	if err != nil {
		return nil, fmt.Errorf("error getting ratelimits: %w", err)
	}
	result.Ratelimits = ratelimits

	if response.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("invalid status code (%d): %s", response.StatusCode, mustString(response))
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return &result, fmt.Errorf("error decoding response: %w", err)
	}

	return &result, nil
}

type StallResponse struct {
	Stall
	Ratelimits Ratelimits
}

func (api *CSFloat) Stall(apiKey, steamId string) (*StallResponse, error) {
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

	ratelimits, err := RatelimitsFrom(response)
	if err != nil {
		return nil, fmt.Errorf("error getting ratelimits: %w", err)
	}

	result := &StallResponse{
		Ratelimits: ratelimits,
	}
	if response.StatusCode != 200 {
		return result, fmt.Errorf("bad response: %d: %s", response.StatusCode, mustString(response))
	}

	if err := json.NewDecoder(response.Body).Decode(&result.Stall); err != nil {
		return result, fmt.Errorf("error decoding response: %w", err)
	}

	return result, nil
}

func mustString(response *http.Response) string {
	b, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	return string(b)
}

type InventoryResponse struct {
	Data       []InventoryItem
	Ratelimits Ratelimits
}

func (api *CSFloat) Inventory(apiKey string) (*InventoryResponse, error) {
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

	ratelimits, err := RatelimitsFrom(response)
	if err != nil {
		return nil, fmt.Errorf("error getting ratelimits: %w", err)
	}

	result := &InventoryResponse{
		Ratelimits: ratelimits,
	}

	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("invalid return code: %d", response.StatusCode)
	}

	// It returns everything, including what's in stall.
	// tradeable will always be 1, since steam does not shot untradable items
	// anymore. Items with a `listing_id` are already in your stall.

	if err := json.NewDecoder(response.Body).Decode(&result.Data); err != nil {
		return result, fmt.Errorf("error decoding response: %w", err)
	}

	return result, nil
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
	AuctionType ListingType `json:"type"`
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
	// Queued means it was just bought, step 1.
	Queued TradeState = "queued"
	// Pending is the stage after queued, meaning the sale was accepted, but
	// has not been verified yet. You can also be in this state if the item
	// is still in trade protection, but would otherwise be verified. In this
	// state it is relevant to check for all the different timestamp fields on
	// the trade object. The verification mode will also already be escrow at
	// this point in time.
	Pending TradeState = "pending"
	// Verified means both sides have received the goods.
	Verified TradeState = "verified"
	// Cancelled means the buyer decided not to buy afterall.
	Cancelled TradeState = "cancelled"
	// Failed means the buyer failed to accept.
	Failed TradeState = "failed"
)

type VerificationMode string

const (
	// Inventory is the mode right after purchase, before anyone has
	// accepted anything until the seller accepted the sale.
	Inventory VerificationMode = "inventory"
	// Escrow means both sides have accepted everything, but the balance isn't
	// usable yet.
	Escrow VerificationMode = "escrow"
)

type Trade struct {
	ID string `json:"id"`
	// BuyerId is the steam ID, which can be your own ID if you are the buyer.
	BuyerId  string     `json:"buyer_id"`
	Contract ListedItem `json:"contract"`
	// CreatedAt is the time at which the sale was made, either through a buy
	// order or manually.
	CreatedAt time.Time `json:"created_at"`
	// AcceptedAt, is the time where the trade accepted the trade on CSFloat.
	AcceptedAt time.Time `json:"accepted_at"`
	// TradeProtectionEndsAt is the time at which the Steam trade protection
	// ends. Only after this, we can verify.
	TradeProtectionEndsAt time.Time `json:"trade_protection_ends_at"`
	// VerifySaleAt is the time after which the traede protection runs out.
	VerifySaleAt time.Time `json:"verify_sale_at"`
	// VerifiedAt is the time at which escrow ended.
	VerifiedAt       time.Time        `json:"verified_at"`
	State            TradeState       `json:"state"`
	VerificationMode VerificationMode `json:"verification_mode"`
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
	Price  uint      `json:"price"`
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

	ratelimits, err := RatelimitsFrom(response)
	if err != nil {
		return nil, fmt.Errorf("error getting ratelimits: %w", err)
	}

	result := &HistoryResponse{
		Ratelimits: ratelimits,
	}

	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("invalid return code: %d", response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(&result.Data); err != nil {
		return result, fmt.Errorf("error decoding response: %w", err)
	}

	return result, nil
}

type BuyResponse struct {
	Ratelimits Ratelimits
	Message    string `json:"message"`
}

type BuyRequestPayload struct {
	ContractIds []string `json:"contract_ids"`
	TotalPrice  uint     `json:"total_price"`
}

func (api *CSFloat) Buy(apiKey string, payload BuyRequestPayload) (*BuyResponse, error) {
	endpoint := "https://csfloat.com/api/v1/listings/buy"
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(payload); err != nil {
		return nil, fmt.Errorf("error encoding payload: %w", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		endpoint,
		&buffer)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	ratelimits, err := RatelimitsFrom(response)
	if err != nil {
		return nil, fmt.Errorf("error getting ratelimits: %w", err)
	}

	result := &BuyResponse{
		Ratelimits: ratelimits,
	}

	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("invalid return code: %d", response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("error decoding response: %w", err)
	}

	return result, nil
}

type ItemBuyOrdersResponse struct {
	Ratelimits Ratelimits
	Data       []ItemBuyOrder
}

type ItemBuyOrder struct {
	// MarketHashName is only used for simple buy orders.
	MarketHashName string `json:"market_hash_name"`
	// Expression is only used for advanced buy orders.
	Expression string `json:"expression"`
	Quantity   uint   `json:"qty"`
	Price      uint   `json:"price"`
}

func (api *CSFloat) ItemBuyOrders(apiKey, listingId string) (*ItemBuyOrdersResponse, error) {
	endpoint := "https://csfloat.com/api/v1/listings/%s/buy-orders?limit=10"
	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprint(endpoint, listingId),
		nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	ratelimits, err := RatelimitsFrom(response)
	if err != nil {
		return nil, fmt.Errorf("error getting ratelimits: %w", err)
	}

	result := &ItemBuyOrdersResponse{
		Ratelimits: ratelimits,
	}

	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("invalid return code: %d", response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(&result.Data); err != nil {
		return result, fmt.Errorf("error decoding response: %w", err)
	}

	return result, nil
}

type SimilarResponse struct {
	Ratelimits Ratelimits
	Data       []*ListedItem
}

func (api *CSFloat) Similar(apiKey, listingId string) (*SimilarResponse, error) {
	endpoint := "https://csfloat.com/api/v1/listings/%s/similar"
	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprint(endpoint, listingId),
		nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	request.Header.Set("Authorization", apiKey)

	response, err := api.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	ratelimits, err := RatelimitsFrom(response)
	if err != nil {
		return nil, fmt.Errorf("error getting ratelimits: %w", err)
	}

	result := &SimilarResponse{
		Ratelimits: ratelimits,
	}

	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("invalid return code: %d", response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(&result.Data); err != nil {
		return result, fmt.Errorf("error decoding response: %w", err)
	}

	return result, nil
}
