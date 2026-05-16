package csfloat

import (
	json "encoding/json/v2"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Fee is a constant fee. Technically the profile has a setting, but it seems
// its unachievable to reduce the fee, so this is fine for now.
const Fee float64 = 2

const (
	ErrorCodeOverpricedRedquireKYC = 4
	ErrorCodeAlreadySold           = 4
	// ErrorCodeInvalidPurchaseState is thrown along HTTP status code 422. It
	// is unclear when exactly, but it seems similar to AlreadySold. It might
	// be unlisted.
	ErrorCodeInvalidPurchaseState = 6
	ErrorCodePriceChanged         = 15
	// ErrorCodeSalesHistoryNotAvailable implies that the history for a certain
	// item was disabled. This is done for cases for example.
	ErrorCodeSalesHistoryNotAvailable = 200
)

type API struct {
	httpClient *http.Client
	apiKey     string
	ratelimits map[RatelimitBucketKey]*Ratelimits
}

func NewWithHTTPClient(apiKey string, client *http.Client) *API {
	return &API{
		httpClient: client,
		apiKey:     apiKey,
		ratelimits: make(map[RatelimitBucketKey]*Ratelimits),
	}
}

func New(apiKey string) *API {
	dialer := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 90 * time.Second,
	}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          2,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 3 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}
	return NewWithHTTPClient(apiKey, client)
}

type Stall struct {
	Items      []ActiveListing `json:"data"`
	Count      int             `json:"total_count"`
	TotalPrice uint            `json:"total_price"`
}

type ListingType string

const (
	BuyNow  ListingType = "buy_now"
	Auction ListingType = "auction"
)

type Rarity uint8

const (
	Consumer   Rarity = 1
	Industrial Rarity = 2
	MilSpec    Rarity = 3
	Restricted Rarity = 4
	Classified Rarity = 5
	Covert     Rarity = 6
	Contraband Rarity = 7
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
	BasePrice     int     `json:"base_price,omitzero"`
	FloatFactor   float64 `json:"float_factor,omitzero"`
	KeyChainPrice int     `json:"keychain_price,omitzero"`
	// PredictedPrice = (BasePrice * FloatFactor) + KeyChainPrice
	PredictedPrice int  `json:"predicted_price,omitzero"`
	Quantity       uint `json:"quantity,omitzero"`
}

// Reference is used as a price reference for items without dynamic factors, such as stickers.
type Reference struct {
	Price    uint `json:"price,omitzero"`
	Quantity uint `json:"quantity,omitzero"`
}

type ListingState string

const (
	ListingStateListed   ListingState = "listed"
	ListingStateRefunded ListingState = "refunded"

	// Note, these are NOT all possible listing states. There are others, but we don't know about them.
)

// Contract is a snapshot of the listing at the time of purchase (or after failing / cancelling)
type Contract struct {
	ID        string        `json:"id"`
	CreatedAt time.Time     `json:"created_at"`
	Price     int           `json:"price"`
	Item      Item          `json:"item"`
	Reference ItemReference `json:"reference,omitzero"`
	Type      ListingType   `json:"type"`
	State     ListingState  `json:"state"`
}

type Seller struct {
	// Obsfuscated ID of the seller
	ObfuscatedID string `json:"obfuscated_id,omitempty"`
	// Steam ID of the seller
	SteamID string `json:"steam_id,omitempty"`
}

type ActiveListing struct {
	ID               string        `json:"id"`
	CreatedAt        time.Time     `json:"created_at"`
	Price            int           `json:"price"`
	Item             Item          `json:"item"`
	Reference        ItemReference `json:"reference,omitzero"`
	Type             ListingType   `json:"type"`
	State            ListingState  `json:"state"`
	Seller           Seller        `json:"seller"`
	Description      string        `json:"description,omitempty"`
	Private          bool          `json:"private,omitzero"`
	MaxOfferDiscount uint          `json:"max_offer_discount,omitzero"`
	Watchers         uint          `json:"watchers,omitzero"`
}

func (al *ActiveListing) URL() string {
	return "https://csfloat.com/item/" + al.ID
}

type InventoryItem struct {
	Item
	// ListingID is only filled for items that are already in the stall.
	ListingID string        `json:"listing_id,omitempty"`
	Reference ItemReference `json:"reference,omitzero"`
}

type Sticker struct {
	Index     uint      `json:"stickerId"`
	Name      string    `json:"name"`
	Reference Reference `json:"reference,omitzero"`
	IconURL   string    `json:"icon_url"`
	Wear      float32   `json:"wear,omitzero"`
	Rotation  float32   `json:"rotation,omitzero"`
}

type Charm struct {
	// CharmId is called stickerId, not a typo.
	Index          uint      `json:"stickerId"`
	Pattern        uint      `json:"pattern,omitzero"`
	WrappedSticker uint      `json:"wrapped_sticker,omitzero"`
	Name           string    `json:"name"`
	IconURL        string    `json:"icon_url"`
	Reference      Reference `json:"reference,omitzero"`
}

type ItemType string

const (
	TypeCharm     ItemType = "charm"
	TypeContainer ItemType = "container"
	TypeSticker   ItemType = "sticker"
	TypeSkin      ItemType = "skin"
	TypeAgent     ItemType = "agent"
)

type Fade struct {
	// Seed sems to be the same as the paintseed so far.
	Seed       uint    `json:"seed"`
	Percentage float64 `json:"percentage"`
	Rank       uint    `json:"rank"`
	Type       string  `json:"type"`
}

type BlueGem struct {
	PlaysideBlue   float64 `json:"playside_blue,omitzero"`
	PlaysideGold   float64 `json:"playside_gold,omitzero"`
	PlaysidePurple float64 `json:"playside_purple,omitzero"`
	BacksideBlue   float64 `json:"backside_blue,omitzero"`
	BacksideGold   float64 `json:"backside_gold,omitzero"`
	BacksidePurple float64 `json:"backside_purple,omitzero"`
}

type Item struct {
	ID             string   `json:"asset_id"`
	Rarity         Rarity   `json:"rarity"`
	Type           ItemType `json:"type"`
	MarketHashName string   `json:"market_hash_name"`
	IconURL        string   `json:"icon_url"`

	// SerializedInspect is the new format for inspect links, replacing InspectLink.
	// InspectLink is still available on float, but not needed in the endpoints. It
	// was previously used for getting concrete buy orders.
	SerializedInspect string  `json:"serialized_inspect,omitempty"`
	Sig               string  `json:"gs_sig,omitempty"`
	ScreenshotID      string  `json:"cs2_screenshot_id,omitempty"`
	Float             float64 `json:"float_value,omitzero"`
	// These two are part of the name and can be retrieved via category.
	// IsStattrak        bool    `json:"is_stattrak,omitempty"`
	// IsSouvenir        bool    `json:"is_souvenir,omitempty"`
	// DefIndex is the weapon type
	DefIndex     uint `json:"def_index,omitzero"`
	StickerIndex uint `json:"sticker_index,omitzero"`
	// PaintIndex is the skin type
	PaintIndex uint `json:"paint_index,omitzero"`
	// PaintSeed determines the skin pattern
	PaintSeed  uint      `json:"paint_seed,omitzero"`
	Stickers   []Sticker `json:"stickers,omitempty"`
	Charms     []Charm   `json:"keychains,omitempty"`
	Fade       *Fade     `json:"fade,omitempty"`
	BlueGem    *BlueGem  `json:"blue_gem,omitempty"`
	Collection string    `json:"collection,omitempty"`

	CharmIndex         uint `json:"keychain_index,omitzero"`
	CharmPattern       uint `json:"keychain_pattern,omitzero"`
	CharmHighlightReel uint `json:"keychain_highlight_reel,omitzero"`
}

func (item *Item) ScreenshotURL(playside bool) string {
	if item.ScreenshotID == "" {
		return ""
	}

	if playside {
		return "https://csfloat.pics/m/" + item.ScreenshotID + "/playside.png?v=3"
	}
	return "https://csfloat.pics/m/" + item.ScreenshotID + "/backside.png?v=3"
}

// Category will map to the query category matching this item. This is required
// for listing similar items.
func (item *Item) Category() Category {
	if strings.HasPrefix(item.MarketHashName, "Souvenir") {
		return Souvenir
	}
	if strings.HasPrefix(item.MarketHashName, "StatTrak") {
		return StatTrak
	}
	return Normal
}

type Category uint

const (
	Normal   = 1
	StatTrak = 2
	Souvenir = 3
)

type SortListingsBy string

const (
	BestDeals       = ""
	Newest          = "most_recent"
	HighestDiscount = "highest_discount"
	LowestPrice     = "lowest_price"
	HighestPrice    = "highest_price"
)

type ListingsRequest struct {
	MinPrice int
	MaxPrice int
	MinFloat float32
	MaxFloat float32
	//ExcludeRare true causes min_ref_qty to be set to 20, just like on the CSFloat page.
	ExcludeRare        bool
	MinRefQuantity     uint
	Category           Category
	SortBy             SortListingsBy
	DefIndex           uint
	StickerIndex       uint
	PaintIndex         uint
	PaintSeed          []uint
	CharmIndex         uint
	CharmHighlightReel uint
	Type               ListingType
}

type ListingResponse struct {
	GenericResponse
	Item ActiveListing
}

func (response *ListingResponse) responseBody() any {
	return &response.Item
}

// Listing returns an existing listing.
func (api *API) Listing(listingId string) (*ListingResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyGetListing,
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/listings/"+listingId,
		api.apiKey,
		nil,
		nil,
		&ListingResponse{},
	)
}

type StallResponse struct {
	GenericResponse
	Stall
}

func (response *StallResponse) responseBody() any {
	return &response.Stall
}

func (api *API) Stall(steamId string) (*StallResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyGetStall,
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/users/"+steamId+"/stall",
		api.apiKey,
		nil,
		url.Values{
			"limit": []string{"40"},
		},
		&StallResponse{},
	)
}

type InventoryResponse struct {
	GenericResponse
	Data []InventoryItem
}

func (response *InventoryResponse) responseBody() any {
	return &response.Data
}

// Inventory returns all visible (tradable) items from the Steam inventory.
// This includes items already listed in the stall, those will have a
// `listing_id` set.
func (api *API) Inventory() (*InventoryResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyGetInventory,
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/me/inventory",
		api.apiKey,
		nil,
		url.Values{
			"limit": []string{"40"},
		},
		&InventoryResponse{},
	)
}

type MeUser struct {
	SteamId        string `json:"steam_id"`
	Balance        uint   `json:"balance"`
	PendingBalance uint   `json:"pending_balance"`
}

type MeResponse struct {
	GenericResponse
	User MeUser `json:"user"`
}

func (response *MeResponse) responseBody() any {
	return response
}

func (api *API) Me() (*MeResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyGetMe,
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/me",
		api.apiKey,
		nil,
		nil,
		&MeResponse{},
	)
}

type PostNewOfferRequest struct {
	GivenAssetIds []string `json:"given_asset_ids"`
	// ReceivedAssetIds is normally empty, as you usually only send assets.
	ReceivedAssetIds []string `json:"received_asset_ids"`
	// OfferId, figure out where it comes from.
	OfferId string `json:"offer_id"`
}

func (api *API) PostNewOffer(offer PostNewOfferRequest) (*GenericResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyPostNewOffer,
		api.httpClient,
		http.MethodPost,
		"https://csfloat.com/api/v1/trades/steam-status/new-offer",
		api.apiKey,
		offer,
		nil,
		&GenericResponse{},
	)
}

type AcceptTradesResponse struct {
	GenericResponse
	Data []Trade `json:"data"`
}

func (response *AcceptTradesResponse) responseBody() any {
	return response
}

func (api *API) BulkAcceptTrade(tradeIds ...string) (*AcceptTradesResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyBulkAcceptTrade,
		api.httpClient,
		http.MethodPost,
		"https://csfloat.com/api/v1/trades/bulk/accept",
		api.apiKey,
		map[string]any{
			"trade_ids": tradeIds,
		},
		nil,
		&AcceptTradesResponse{},
	)
}

func (api *API) BulkCancel(tradeIds ...string) (*GenericResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyBulkCancel,
		api.httpClient,
		http.MethodPost,
		"https://csfloat.com/api/v1/trades/bulk/cancel",
		api.apiKey,
		map[string]any{
			"trade_ids": tradeIds,
		},
		nil,
		&GenericResponse{},
	)
}

type BulkListRequest struct {
	Items []ListRequest `json:"items"`
}

type BulkListResponse struct {
	GenericResponse
	Data []ActiveListing `json:"data"`
}

func (response *BulkListResponse) responseBody() any {
	return response
}

func (api *API) BulkList(items ...ListRequest) (*BulkListResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyBulkList,
		api.httpClient,
		http.MethodPost,
		"https://csfloat.com/api/v1/listings/bulk-list",
		api.apiKey,
		BulkListRequest{Items: items},
		nil,
		&BulkListResponse{},
	)
}

func (api *API) BulkUnlist(listingId ...string) (*GenericResponse, error) {
	if len(listingId) == 0 {
		return nil, errors.New("no listings supplied")
	}
	if len(listingId) == 1 {
		return handleRequest(
			api,
			RatelimitKeyUnlist,
			api.httpClient,
			http.MethodDelete,
			"https://csfloat.com/api/v1/listings/"+listingId[0],
			api.apiKey,
			nil,
			nil,
			&GenericResponse{},
		)
	}

	return handleRequest(
		api,
		RatelimitKeyBulkUnlist,
		api.httpClient,
		http.MethodPatch,
		"https://csfloat.com/api/v1/listings/bulk-delist",
		api.apiKey,
		map[string][]string{
			"contract_ids": listingId,
		},
		nil,
		&GenericResponse{},
	)
}

type UnlistResponse struct {
	GenericResponse
}

func (response *UnlistResponse) responseBody() any {
	return nil
}

func (api *API) Unlist(listingId string) (*UnlistResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyUnlist,
		api.httpClient,
		http.MethodDelete,
		"https://csfloat.com/api/v1/listings/"+listingId,
		api.apiKey,
		nil,
		nil,
		&UnlistResponse{},
	)
}

type UpdateListingResponse struct {
	ListingResponse
}

func (api *API) UpdatePrivate(listingId string, private bool) (*UpdateListingResponse, error) {
	return api.updateListing(listingId, map[string]any{"private": private})
}

func (api *API) UpdateDescription(listingId string, description string) (*UpdateListingResponse, error) {
	return api.updateListing(listingId, map[string]any{"description": description})
}

func (api *API) UpdateDiscount(listingId string, discount uint) (*UpdateListingResponse, error) {
	return api.updateListing(listingId, map[string]any{"max_offer_discount": discount})
}

func (api *API) UpdatePrice(listingId string, price uint) (*UpdateListingResponse, error) {
	return api.updateListing(listingId, map[string]any{"price": price})
}

type UpdateListingRequest struct {
	Private          bool   `json:"private"`
	Description      string `json:"description"`
	MaxOfferDiscount uint   `json:"max_offer_discount"`
	Price            uint   `json:"price"`
}

func (api *API) UpdateListing(id string, payload UpdateListingRequest) (*UpdateListingResponse, error) {
	return api.updateListing(id, payload)
}

func (api *API) updateListing(listingId string, payload any) (*UpdateListingResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyUpdateListing,
		api.httpClient,
		http.MethodPatch,
		"https://csfloat.com/api/v1/listings/"+listingId,
		api.apiKey,
		payload,
		nil,
		&UpdateListingResponse{},
	)
}

type BuyNowRequest struct {
	Price uint `json:"price,omitzero"`
}

type AuctionRequest struct {
	DurationDays uint `json:"duration_days,omitzero"`
	ReservePrice uint `json:"reserve_price,omitzero"`
}

type ListRequest struct {
	*BuyNowRequest
	*AuctionRequest

	AssetId     string      `json:"asset_id"`
	AuctionType ListingType `json:"type"`
	Description string      `json:"description,omitempty"`
	Private     bool        `json:"private,omitzero"`
}

type Error struct {
	HttpStatus uint   `json:"-"`
	Code       uint   `json:"code"`
	Message    string `json:"message"`
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

type SteamOfferState int

const (
	NotSent                         SteamOfferState = 0
	SentAwaitingMobileAuthenticator SteamOfferState = 9
	Accepted                        SteamOfferState = 3
)

type SteamOffer struct {
	State  int       `json:"state"`
	SentAt time.Time `json:"sent_at"`
}

type Trade struct {
	ID string `json:"id"`
	// BuyerId is the steam ID, which can be your own ID if you are the buyer.
	BuyerId  string   `json:"buyer_id"`
	Contract Contract `json:"contract"`
	// CreatedAt is the time at which the sale was made, either through a buy
	// order or manually.
	CreatedAt time.Time `json:"created_at"`
	// AcceptedAt, is the time where the trade accepted the trade on CSFloat.
	AcceptedAt time.Time `json:"accepted_at,omitzero"`
	// CURRENTLY UNUSED
	// SteamOffer SteamOffer `json:"steam_offer"`
	// TradeProtectionEndsAt is the time at which the Steam trade protection
	// ends. Only after this, we can verify.
	TradeProtectionEndsAt time.Time `json:"trade_protection_ends_at,omitzero"`
	// VerifySaleAt is the time after which the traede protection runs out.
	VerifySaleAt time.Time `json:"verify_sale_at,omitzero"`
	// VerifiedAt is the time at which escrow ended.
	VerifiedAt       time.Time        `json:"verified_at,omitzero"`
	State            TradeState       `json:"state"`
	VerificationMode VerificationMode `json:"verification_mode,omitempty"`
}

type TradesResponse struct {
	GenericResponse
	Trades []Trade `json:"trades"`
	// Count is the total count of trades (all pages for the given query)
	Count uint `json:"count"`
}

func (response *TradesResponse) responseBody() any {
	return response
}

type TradesRequest struct {
	// Page, default 0 (latest)
	Page uint `json:"page"`
	// Limit, default 100
	Limit uint `json:"limit"`
	// States, empty by default, not filtering
	States []TradeState
}

func (api *API) Trades(payload TradesRequest) (*TradesResponse, error) {
	if payload.Limit == 0 {
		payload.Limit = 100
	}

	form := url.Values{}
	if len(payload.States) > 0 {
		var str strings.Builder
		// longest state is cancelled with 9, so we'd fit cancelled + a comma
		str.Grow(len(payload.States) * 10)
		for index, state := range payload.States {
			str.WriteString(string(state))
			if index < len(payload.States)-1 {
				str.WriteByte(',')
			}
		}
		form.Set("state", str.String())
	}
	form.Set("page", strconv.FormatUint(uint64(payload.Page), 10))
	form.Set("limit", strconv.FormatUint(uint64(payload.Limit), 10))

	return handleRequest(
		api,
		RatelimitKeyGetTrades,
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/me/trades",
		api.apiKey,
		nil,
		form,
		&TradesResponse{},
	)
}

type HistoryEntry struct {
	Price     uint          `json:"price"`
	Item      Item          `json:"item"`
	Reference ItemReference `json:"reference,omitzero"`
	SoldAt    time.Time     `json:"sold_at"`
}

type HistoryRequestPayload struct {
	MarketHashName string
	PaintIndex     uint
}

type HistoryResponse struct {
	GenericResponse
	Data []HistoryEntry
}

func (response *HistoryResponse) responseBody() any {
	return &response.Data
}

func (api *API) History(payload HistoryRequestPayload) (*HistoryResponse, error) {
	form := url.Values{}

	// Passing zero for a case / sticker will yield in an empty result.
	// paint_index 0 for weapons is supposedly the default skin.
	if payload.PaintIndex != 0 {
		form.Set("paint_index", strconv.FormatUint(uint64(payload.PaintIndex), 10))
	}

	return handleRequest(
		api,
		RatelimitKeyGetHistory,
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/history/"+url.QueryEscape(payload.MarketHashName)+"/sales",
		api.apiKey,
		nil,
		form,
		&HistoryResponse{},
	)
}

type BuyResponse struct {
	GenericResponse
}

func (response *BuyResponse) responseBody() any {
	return nil
}

type BuyRequestPayload struct {
	ContractIds []string `json:"contract_ids"`
	TotalPrice  uint     `json:"total_price"`
}

func (api *API) Buy(payload BuyRequestPayload) (*BuyResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyBuy,
		api.httpClient,
		http.MethodPost,
		"https://csfloat.com/api/v1/listings/buy",
		api.apiKey,
		payload,
		nil,
		&BuyResponse{},
	)
}

func (api *API) Unwatch(listingId string) (*GenericResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyUnwatch,
		api.httpClient,
		http.MethodDelete,
		"https://csfloat.com/api/v1/listings/"+listingId+"/watchlist",
		api.apiKey,
		nil,
		nil,
		&GenericResponse{},
	)
}

func (api *API) Watch(listingId string) (*GenericResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyWatch,
		api.httpClient,
		http.MethodPost,
		"https://csfloat.com/api/v1/listings/"+listingId+"/watchlist",
		api.apiKey,
		nil,
		nil,
		&GenericResponse{},
	)
}

// SimpleItemBuyOrdersResponse is the same as BuyOrders content-wise, BUT the structure is different.
type SimpleItemBuyOrdersResponse struct {
	GenericResponse
	Data []ItemBuyOrder `json:"data"`
}

func (response *SimpleItemBuyOrdersResponse) responseBody() any {
	return response
}

type ItemBuyOrdersResponse struct {
	GenericResponse
	Data []ItemBuyOrder `json:"data"`
}

func (response *ItemBuyOrdersResponse) responseBody() any {
	return &response.Data
}

type ItemBuyOrder struct {
	ID string `json:"id"`
	// MarketHashName is only used for simple buy orders.
	MarketHashName string `json:"market_hash_name"`
	// Expression is only used for advanced buy orders.
	Expression string `json:"expression,omitempty"`
	Quantity   uint   `json:"qty,omitzero"`
	Price      uint   `json:"price"`
}

func (api *API) ItemBuyOrders(item *Item) (*ItemBuyOrdersResponse, error) {
	formValues := url.Values{"limit": []string{"3"}}

	formValues.Set("url", item.SerializedInspect)
	formValues.Set("sig", item.Sig)
	formValues.Set("market_hash_name", item.MarketHashName)

	method := http.MethodGet
	url := "https://csfloat.com/api/v1/buy-orders/item"

	return handleRequest(
		api,
		RatelimitKeyGetItemBuyOrders,
		api.httpClient,
		method,
		url,
		api.apiKey,
		nil,
		formValues,
		&ItemBuyOrdersResponse{},
	)
}

func (api *API) SimpleItemBuyOrders(item *Item) (*SimpleItemBuyOrdersResponse, error) {
	formValues := url.Values{"limit": []string{"3"}}

	body := map[string]string{
		"market_hash_name": item.MarketHashName,
	}
	// Well ... this is being abused for providing a body it seems.
	method := http.MethodPost
	url := "https://csfloat.com/api/v1/buy-orders/similar-orders"

	return handleRequest(
		api,
		RatelimitKeyGetSimpleItemBuyOrders,
		api.httpClient,
		method,
		url,
		api.apiKey,
		body,
		formValues,
		&SimpleItemBuyOrdersResponse{},
	)
}

func (api *API) ListingBuyOrders(listingId string, limit int64) (*ItemBuyOrdersResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyGetListingBuyOrders,
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/listings/"+listingId+"/buy-orders",
		api.apiKey,
		nil,
		url.Values{"limit": []string{strconv.FormatInt(limit, 10)}},
		&ItemBuyOrdersResponse{},
	)
}

type SimilarResponse struct {
	GenericResponse
	Data []*ActiveListing
}

func (response *SimilarResponse) responseBody() any {
	return &response.Data
}

func (api *API) Similar(listingId string) (*SimilarResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyGetSimilar,
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/listings/"+listingId+"/similar",
		api.apiKey,
		nil,
		nil,
		&SimilarResponse{},
	)
}

func errorFrom(response *http.Response) (Error, error) {
	var csfloatError Error
	csfloatError.HttpStatus = uint(response.StatusCode)
	return csfloatError, json.UnmarshalRead(response.Body, &csfloatError)
}

type TransactionType string

const (
	TransactionTypeDeposit                TransactionType = "deposit"
	TransactionTypeWithdrawal             TransactionType = "withdrawal"
	TransactionTypeContractSold           TransactionType = "contract_sold"
	TransactionTypeContractSaleRefund     TransactionType = "contract_sale_refund"
	TransactionTypeContractPurchased      TransactionType = "contract_purchased"
	TransactionTypeContractPurchaseRefund TransactionType = "contract_purchase_refund"
	TransactionTypeTradeVerified          TransactionType = "trade_verified"
	TransactionTypeFine                   TransactionType = "fine"
	TransactionTypeBidDeclined            TransactionType = "bid_declined"
	TransactionTypeBidPosted              TransactionType = "bid_posted"
)

type TransactionDetailType string

const (
	TransactionDetailTypeBuyerConfirm TransactionDetailType = "buyer_confirm"
	TransactionDetailTypeBuyerPing    TransactionDetailType = "buyer_ping"
	TransactionDetailTypeLink         TransactionDetailType = "link"
	TransactionDetailTypeFloatDB      TransactionDetailType = "floatdb"
)

type TransactionDetails struct {
	ContractID string `json:"contract_id,omitempty"`
	TradeID    string `json:"trade_id,omitempty"`
	BidID      string `json:"bid_id,omitempty"`
	// ListingID is used for BidPosted
	ListingID             string                `json:"listing_id,omitempty"`
	BuyOrderID            string                `json:"buy_order_id,omitempty"`
	OriginalTransactionId string                `json:"original_tx,omitempty"`
	Type                  TransactionDetailType `json:"type,omitempty"`
	// FeeAmountString should not be used, use the FeeAmount function instead.
	FeeAmountString string `json:"fee_amount,omitempty"`
	// Reason is used for fines and others.
	Reason string `json:"reason,omitempty"`

	// Fee is used for Deposits. God knows why its a seperate field and not
	// FeeAmountString.
	FeeString        string `json:"fee,omitempty"`
	PaymentMethod    string `json:"payment_method,omitempty"`
	PaymentProcessor string `json:"payment_processor,omitempty"`
	// SessionID is for strip deposits.
	SessionID string `json:"session_id,omitempty"`

	// Withdrawal Fields
	FloatFeeString string `json:"float_fee,omitempty"`
}

func (details TransactionDetails) FloatFee() int {
	if details.FloatFeeString == "" {
		return 0
	}

	i, _ := strconv.ParseInt(details.FloatFeeString, 10, 32)
	return int(i)
}

func (details TransactionDetails) Fee() int {
	if details.FeeString == "" {
		return 0
	}

	i, _ := strconv.ParseInt(details.FeeString, 10, 32)
	return int(i)
}

func (details TransactionDetails) FeeAmount() int {
	if details.FeeAmountString == "" {
		return 0
	}

	i, _ := strconv.ParseInt(details.FeeAmountString, 10, 32)
	return int(i)
}

type Transaction struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	// UserID seems to be our SteamID and is always the same ID.
	// UserID        string             `json:"user_id"`
	Type          TransactionType    `json:"type"`
	Details       TransactionDetails `json:"details"`
	BalanceOffset int                `json:"balance_offset,omitzero"`
	PendingOffset int                `json:"pending_offset,omitzero"`
}

type TransactionsResponse struct {
	GenericResponse
	Transactions []Transaction `json:"transactions"`
	Count        uint          `json:"count"`
}

type Order string

const (
	OrderDesc Order = "desc"
	OrderAsc  Order = "asc"
)

type TransactionsRequest struct {
	// Page, default 0 (latest)
	Page uint
	// Limit, default 100
	Limit uint
	Order Order
}

func (response *TransactionsResponse) responseBody() any {
	return response
}

func (api *API) Transactions(payload TransactionsRequest) (*TransactionsResponse, error) {
	if payload.Limit == 0 {
		payload.Limit = 100
	}

	form := url.Values{}
	form.Set("order", "desc")
	form.Set("page", strconv.FormatUint(uint64(payload.Page), 10))
	form.Set("limit", strconv.FormatUint(uint64(payload.Limit), 10))

	return handleRequest(
		api,
		RatelimitKeyGetTransactions,
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/me/transactions",
		api.apiKey,
		nil,
		form,
		&TransactionsResponse{},
	)
}

type ListResponse struct {
	GenericResponse
	Item ActiveListing
}

func (response *ListResponse) responseBody() any {
	return &response.Item
}

func (api *API) List(payload ListRequest) (*ListResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyCreateListing,
		api.httpClient,
		http.MethodPost,
		"https://csfloat.com/api/v1/listings",
		api.apiKey,
		payload,
		nil,
		&ListResponse{},
	)
}

type ListingsResponse struct {
	GenericResponse
	Data []ActiveListing `json:"data"`
}

func (response *ListingsResponse) responseBody() any {
	return response
}

func (api *API) Listings(query ListingsRequest) (*ListingsResponse, error) {
	form := url.Values{}
	form.Set("limit", "40")
	// Empty = BestDeals = Default
	if query.SortBy != BestDeals {
		form.Set("sort_by", string(query.SortBy))
	}
	if query.MinRefQuantity > 0 {
		form.Set("min_ref_qty", strconv.FormatUint(uint64(query.MinRefQuantity), 10))
	} else if query.ExcludeRare {
		form.Set("min_ref_qty", strconv.FormatUint(20, 10))
	}
	if query.DefIndex > 0 {
		form.Set("def_index", strconv.FormatUint(uint64(query.DefIndex), 10))
	}
	if query.PaintIndex > 0 {
		form.Set("paint_index", strconv.FormatUint(uint64(query.PaintIndex), 10))
	}
	if len(query.PaintSeed) > 0 {
		form.Set("paint_seed", concatInts(query.PaintSeed...))
	}
	if query.Category > 0 {
		form.Set("category", strconv.FormatUint(uint64(query.Category), 10))
	}
	if query.MinPrice > 0 {
		form.Set("min_price", strconv.Itoa(query.MinPrice))
	}
	if query.MaxPrice > 0 {
		form.Set("max_price", strconv.Itoa(query.MaxPrice))
	}
	if query.MinFloat > 0 {
		form.Set("min_float", strconv.FormatFloat(float64(query.MinFloat), 'f', -1, 32))
	}
	if query.MaxFloat > 0 {
		form.Set("max_float", strconv.FormatFloat(float64(query.MaxFloat), 'f', -1, 32))
	}
	if query.StickerIndex > 0 {
		form.Set("sticker_index", strconv.FormatUint(uint64(query.StickerIndex), 10))
	}
	if query.CharmIndex > 0 {
		form.Set("keychain_index", strconv.FormatUint(uint64(query.CharmIndex), 10))
	}
	if query.CharmHighlightReel > 0 {
		form.Set("keychain_highlight_reel", strconv.FormatUint(uint64(query.CharmHighlightReel), 10))
	}
	if query.Type != "" {
		form.Set("type", string(query.Type))
	}

	return handleRequest(
		api,
		RatelimitKeyGetListings,
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/listings",
		api.apiKey,
		nil,
		form,
		&ListingsResponse{},
	)
}

type CreateSimpleBuyOrderPayload struct {
	MarketHashName string `json:"market_hash_name"`
	MaxPrice       uint   `json:"max_price"`
	Quantity       uint   `json:"quantity"`
}

type CreateSimpleBuyOrderResponse struct {
	GenericResponse
	ItemBuyOrder
}

func (response *CreateSimpleBuyOrderResponse) responseBody() any {
	return response
}

func (api *API) CreateSimpleBuyOrder(payload CreateSimpleBuyOrderPayload) (*CreateSimpleBuyOrderResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyCreateBuyOrder,
		api.httpClient,
		http.MethodPost,
		"https://csfloat.com/api/v1/buy-orders",
		api.apiKey,
		payload,
		nil,
		&CreateSimpleBuyOrderResponse{},
	)
}

func (api *API) DeleteBuyOrder(id string) (*GenericResponse, error) {
	return handleRequest(
		api,
		RatelimitKeyDeleteBuyOrder,
		api.httpClient,
		http.MethodDelete,
		"https://csfloat.com/api/v1/buy-orders/"+id,
		api.apiKey,
		nil,
		nil,
		&GenericResponse{},
	)
}
