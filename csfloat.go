package csfloat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	ErrorCodeAlreadySold = 4
	// ErrorCodeInvalidPurchaseState is thrown along HTTP status code 422. It
	// is unclear when exactly, but it seems similar to AlreadySold. It might
	// be unlisted.
	ErrorCodeInvalidPurchaseState = 6
	ErrorCodePriceChanged         = 15
	// ErrorCodeSalesHistoryNotAvailable implies that the history for a certain
	// item was disabled. This is done for cases for example.
	ErrorCodeSalesHistoryNotAvailable = 200
)

type CSFloat struct {
	httpClient *http.Client
}

func NewWithHTTPClient(client *http.Client) *CSFloat {
	return &CSFloat{
		httpClient: client,
	}
}

func New() *CSFloat {
	dialer := &net.Dialer{
		Timeout: 15 * time.Second,
	}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 3 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}
	return NewWithHTTPClient(client)
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
	BasePrice      int  `json:"base_price"`
	PredictedPrice int  `json:"predicted_price"`
	Quantity       uint `json:"quantity"`
}

type Reference struct {
	Price    uint `json:"price"`
	Quantity uint `json:"quantity"`
}

type ListedItem struct {
	ID               string        `json:"id"`
	Price            int           `json:"price"`
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
	// ListingID is only filled for items that are already in the stall.
	ListingID string        `json:"listing_id"`
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
	IconURL        string   `json:"icon_url"`

	// InspectLink is used to open CS. However, CSFloat also uses it as a key to
	// filter buy orders for a concrete asset.
	InspectLink string  `json:"inspect_link"`
	Float       float64 `json:"float_value"`
	IsStattrak  bool    `json:"is_stattrak"`
	IsSouvenir  bool    `json:"is_souvenir"`
	// DefIndex is the weapon type
	DefIndex uint `json:"def_index"`
	// PaintIndex is the skin type
	PaintIndex uint `json:"paint_index"`
	// PaintSeed determines the skin pattern
	PaintSeed uint      `json:"paint_seed"`
	Stickers  []Sticker `json:"stickers,omitempty"`
	Charms    []Charm   `json:"keychains,omitempty"`
	Fade      *Fade     `json:"fade,omitempty"`
	BlueGem   *BlueGem  `json:"blue_gem,omitempty"`

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

type Category uint

const (
	Normal   = 1
	StatTrak = 2
	Souvenir = 3
)

type ListingsRequest struct {
	MinPrice    int
	MaxPrice    int
	MinFloat    float32
	MaxFloat    float32
	ExcludeRare bool
	Category    Category
	DefIndex    uint
	PaintIndex  uint
	PaintSeed   []uint
	CharmIndex  uint
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

type ListingResponse struct {
	GenericResponse
	Item ListedItem
}

func (response *ListingResponse) responseBody() any {
	return &response.Item
}

// Listing returns an existing listing.
func (api *CSFloat) Listing(apiKey string, listingId string) (*ListingResponse, error) {
	return handleRequest(
		api.httpClient,
		http.MethodGet,
		fmt.Sprintf("https://csfloat.com/api/v1/listings/%s", listingId),
		apiKey,
		nil,
		url.Values{},
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

func (api *CSFloat) Stall(apiKey, steamId string) (*StallResponse, error) {
	return handleRequest(
		api.httpClient,
		http.MethodGet,
		fmt.Sprintf("https://csfloat.com/api/v1/users/%s/stall", steamId),
		apiKey,
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
func (api *CSFloat) Inventory(apiKey string) (*InventoryResponse, error) {
	return handleRequest(
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/me/inventory",
		apiKey,
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

func (api *CSFloat) Me(apiKey string) (*MeResponse, error) {
	return handleRequest(
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/me",
		apiKey,
		nil,
		url.Values{},
		&MeResponse{},
	)
}

type UnlistResponse struct {
	GenericResponse
}

func (response *UnlistResponse) responseBody() any {
	return nil
}

func (api *CSFloat) Unlist(apiKey, listingId string) (*UnlistResponse, error) {
	return handleRequest(
		api.httpClient,
		http.MethodDelete,
		fmt.Sprintf("https://csfloat.com/api/v1/listings/%s", listingId),
		apiKey,
		nil,
		url.Values{},
		&UnlistResponse{},
	)
}

type UpdateListingResponse struct {
	GenericResponse
}

func (response *UpdateListingResponse) responseBody() any {
	return nil
}

type UpdateListingRequest struct {
	MaxOfferDiscount uint `json:"max_offer_discount"`
}

func (api *CSFloat) UpdateListing(apiKey, id string, payload UpdateListingRequest) (*UpdateListingResponse, error) {
	return handleRequest(
		api.httpClient,
		http.MethodPatch,
		fmt.Sprintf("https://csfloat.com/api/v1/listings/%s", id),
		apiKey,
		payload,
		url.Values{},
		&UpdateListingResponse{},
	)
}

type BuyNowRequest struct {
	Price uint `json:"price,omitempty"`
}

type AuctionRequest struct {
	DurationDays uint `json:"duration_days,omitempty"`
	ReservePrice uint `json:"reserve_price,omitempty"`
}

type ListRequest struct {
	*BuyNowRequest
	*AuctionRequest

	AssetId     string      `json:"asset_id"`
	AuctionType ListingType `json:"type"`
	Description string      `json:"description"`
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
	GenericResponse
	Trades []Trade `json:"trades"`
	Count  uint    `json:"count"`
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

func (api *CSFloat) Trades(apiKey string, payload TradesRequest) (*TradesResponse, error) {
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

	return handleRequest(
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/me/trades",
		apiKey,
		nil,
		form,
		&TradesResponse{},
	)
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
	GenericResponse
	Data []HistoryEntry
}

func (response *HistoryResponse) responseBody() any {
	return &response.Data
}

func (api *CSFloat) History(apiKey string, payload HistoryRequestPayload) (*HistoryResponse, error) {
	form := url.Values{}
	form.Set("paint_index", strconv.FormatUint(uint64(payload.PaintIndex), 10))

	return handleRequest(
		api.httpClient,
		http.MethodGet,
		fmt.Sprintf("https://csfloat.com/api/v1/history/%s/sales", url.QueryEscape(payload.MarketHashName)),
		apiKey,
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

func (api *CSFloat) Buy(apiKey string, payload BuyRequestPayload) (*BuyResponse, error) {
	return handleRequest(
		api.httpClient,
		http.MethodPost,
		"https://csfloat.com/api/v1/listings/buy",
		apiKey,
		payload,
		url.Values{},
		&BuyResponse{},
	)
}

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
	// MarketHashName is only used for simple buy orders.
	MarketHashName string `json:"market_hash_name"`
	// Expression is only used for advanced buy orders.
	Expression string `json:"expression"`
	Quantity   uint   `json:"qty"`
	Price      uint   `json:"price"`
}

func (api *CSFloat) ItemBuyOrders(apiKey string, item *Item) (*ItemBuyOrdersResponse, error) {
	formValues := url.Values{"limit": []string{"3"}}

	formValues.Set("url", item.InspectLink)
	method := http.MethodGet
	url := "https://csfloat.com/api/v1/buy-orders/item"

	return handleRequest(
		api.httpClient,
		method,
		url,
		apiKey,
		nil,
		formValues,
		&ItemBuyOrdersResponse{},
	)
}

func (api *CSFloat) SimpleItemBuyOrders(apiKey string, item *Item) (*SimpleItemBuyOrdersResponse, error) {
	formValues := url.Values{"limit": []string{"3"}}

	body := map[string]string{
		"market_hash_name": item.MarketHashName,
	}
	// Well ... this is being abused for providing a body it seems.
	method := http.MethodPost
	url := "https://csfloat.com/api/v1/buy-orders/similar-orders"

	return handleRequest(
		api.httpClient,
		method,
		url,
		apiKey,
		body,
		formValues,
		&SimpleItemBuyOrdersResponse{},
	)
}

func (api *CSFloat) ListingBuyOrders(apiKey, listingId string) (*ItemBuyOrdersResponse, error) {
	return handleRequest(
		api.httpClient,
		http.MethodGet,
		fmt.Sprintf("https://csfloat.com/api/v1/listings/%s/buy-orders", listingId),
		apiKey,
		nil,
		url.Values{"limit": []string{"10"}},
		&ItemBuyOrdersResponse{},
	)
}

type SimilarResponse struct {
	GenericResponse
	Data []*ListedItem
}

func (response *SimilarResponse) responseBody() any {
	return &response.Data
}

func (api *CSFloat) Similar(apiKey, listingId string) (*SimilarResponse, error) {
	return handleRequest(
		api.httpClient,
		http.MethodGet,
		fmt.Sprintf("https://csfloat.com/api/v1/listings/%s/similar", listingId),
		apiKey,
		nil,
		url.Values{},
		&SimilarResponse{},
	)
}

func errorFrom(response *http.Response) (Error, error) {
	var csfloatError Error
	csfloatError.HttpStatus = uint(response.StatusCode)
	return csfloatError, json.NewDecoder(response.Body).Decode(&csfloatError)
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
	ContractID string `json:"contract_id"`
	TradeID    string `json:"trade_id"`
	BidID      string `json:"bid_id"`
	// ListingID is used for BidPosted
	ListingID             string                `json:"listing_id"`
	BuyOrderID            string                `json:"buy_order_id"`
	OriginalTransactionId string                `json:"original_tx"`
	Type                  TransactionDetailType `json:"type"`
	// FeeAmountString should not be used, use the FeeAmount function instead.
	FeeAmountString string `json:"fee_amount"`
	// Reason is used for fines and others.
	Reason string `json:"reason"`

	// Fee is used for Deposits. God knows why its a seperate field and not
	// FeeAmountString.
	FeeString        string `json:"fee"`
	PaymentMethod    string `json:"payment_method"`
	PaymentProcessor string `json:"payment_processor"`
	// SessionID is for strip deposits.
	SessionID string `json:"session_id"`
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
	ID            string             `json:"id"`
	CreatedAt     time.Time          `json:"created_at"`
	UserID        string             `json:"user_id"`
	Type          TransactionType    `json:"type"`
	Details       TransactionDetails `json:"details"`
	BalanceOffset int                `json:"balance_offset"`
	PendingOffset int                `json:"pending_offset"`
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

func (api *CSFloat) Transactions(apiKey string, payload TransactionsRequest) (*TransactionsResponse, error) {
	if payload.Limit == 0 {
		payload.Limit = 100
	}

	form := url.Values{}
	form.Set("order", "desc")
	form.Set("page", strconv.FormatUint(uint64(payload.Page), 10))
	form.Set("limit", strconv.FormatUint(uint64(payload.Limit), 10))

	return handleRequest(
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/me/transactions",
		apiKey,
		nil,
		form,
		&TransactionsResponse{},
	)
}

type ListResponse struct {
	GenericResponse
	Item ListedItem
}

func (response *ListResponse) responseBody() any {
	return &response.Item
}

func (api *CSFloat) List(apiKey string, payload ListRequest) (*ListResponse, error) {
	return handleRequest(
		api.httpClient,
		http.MethodPost,
		"https://csfloat.com/api/v1/listings",
		apiKey,
		payload,
		url.Values{},
		&ListResponse{},
	)
}

type ListingsResponse struct {
	GenericResponse
	Data []ListedItem `json:"data"`
}

func (response *ListingsResponse) responseBody() any {
	return response
}

func (api *CSFloat) Listings(apiKey string, query ListingsRequest) (*ListingsResponse, error) {
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
	if len(query.PaintSeed) > 0 {
		form.Set("paint_seed", concatInts(query.PaintSeed...))
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
	if query.MinFloat > 0 {
		form.Set("min_float", fmt.Sprintf("%0f", query.MinFloat))
	}
	// FIXME Set max float to 0.99 to avoid charms and stickers?
	if query.MaxFloat > 0 {
		form.Set("max_float", fmt.Sprintf("%0f", query.MaxFloat))
	}
	if query.CharmIndex > 0 {
		form.Set("keychain_index", fmt.Sprintf("%d", query.CharmIndex))
	}

	return handleRequest(
		api.httpClient,
		http.MethodGet,
		"https://csfloat.com/api/v1/listings",
		apiKey,
		nil,
		form,
		&ListingsResponse{},
	)
}

type GenericResponse struct {
	// Ratelimits will have zero values if the request fails completly.
	Ratelimits Ratelimits `json:"-"`
	// Error will only be set if an error happened after successfully reaching
	// the server. However, there might still be other errors, for example when
	// decoding the server response.
	Error *Error `json:"-"`
}

func (response *GenericResponse) setRatelimits(ratelimits *Ratelimits) {
	response.Ratelimits = *ratelimits
}
func (response *GenericResponse) setError(err *Error) {
	response.Error = err
}

type Response interface {
	setError(*Error)
	setRatelimits(*Ratelimits)
	// responseBody must return any pointer value that we'll JSON-decode into.
	responseBody() any
}

func handleRequest[T Response](
	client *http.Client,
	method string,
	endpoint string,
	apiKey string,
	payload any,
	form url.Values,
	result T,
) (T, error) {
	var body io.Reader
	var buffer bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&buffer).Encode(payload); err != nil {
			return result, fmt.Errorf("error encoding payload: %w", err)
		}
		body = &buffer
	}

	request, err := http.NewRequest(
		method,
		endpoint,
		body)

	request.URL.RawQuery = form.Encode()

	request.Header.Set("Authorization", apiKey)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Content-Length", strconv.Itoa(buffer.Len()))
	}

	if err != nil {
		return result, fmt.Errorf("error creating request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return result, fmt.Errorf("error sending request: %w", err)
	}

	ratelimits, err := ratelimitsFrom(response)
	if err != nil {
		return result, fmt.Errorf("error getting ratelimits: %w", err)
	}
	result.setRatelimits(&ratelimits)

	if response.StatusCode != http.StatusOK {
		csfloatError, err := errorFrom(response)
		if err != nil {
			return result, fmt.Errorf("invalid status code, couldn't read error message: %d",
				response.StatusCode)
		}
		result.setError(&csfloatError)

		return result, fmt.Errorf("invalid status code: %d; %v", response.StatusCode, csfloatError)
	}

	if target := result.responseBody(); target != nil {
		if err := json.NewDecoder(response.Body).Decode(result.responseBody()); err != nil {
			return result, fmt.Errorf("error decoding response: %w", err)
		}
	}

	return result, nil
}

func concatInts[Number int | uint](n ...Number) string {
	var b strings.Builder
	for i, val := range n {
		if i != 0 {
			b.WriteRune(',')
		}
		b.WriteString(fmt.Sprintf("%d", val))
	}
	return b.String()
}
