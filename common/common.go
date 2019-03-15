package common

import (
	"time"

	"github.com/google/uuid"
)

type PubSubInterface interface {
	Sub(...string) chan interface{}
	Pub(interface{}, ...string)
}

// Event type enums
const (
	EventTypeNewOrder  = "newOrder"
	EventTypeShelved   = "shelved"
	EventTypeReshelved = "reshelved"
	EventTypeWaste     = "waste"
	EventTypePickup    = "pickup"
	EventTypeExpired   = "expired"
	EventTypeValue     = "value"
	EventTypeKeyboard  = "keyboard"
)

// Order data
type Order struct {
	ID        uuid.UUID
	Name      string  `json:"name"`
	Temp      string  `json:"temp"`
	ShelfLife float32 `json:"shelfLife"`
	DecayRate float32 `json:"decayRate"`
}

// NewOrderEvent is fired when a new order arrives in the system
type NewOrderEvent struct {
	Dt    time.Time
	Order Order
}

// ShelvedEvent fires when an order is shelved
type ShelvedEvent struct {
	Dt    time.Time
	Shelf string
	Order Order
}

// ReshelvedEvent fires when an order is moved from overflow to main
type ReshelvedEvent struct {
	Dt      time.Time
	OrderID uuid.UUID
}

// Waste reasons
const (
	WasteReasonExpired      = "expired"
	WasteReasonNoShelfSpace = "noShelfSpace"
)

// WasteEvent fires when an order is declared waste
type WasteEvent struct {
	Dt     time.Time
	Order  Order
	Reason string
}

// PickupEvent dk
type PickupEvent struct {
	Dt    time.Time
	Order Order
}

// ExpiredEvent blah
type ExpiredEvent struct {
	Dt    time.Time
	Order Order
}

// ValueEvent blah
type ValueEvent struct {
	Dt        time.Time
	Shelf     string
	Value     float32
	NormValue float32
	Order     Order
	LastMoved time.Time
}
