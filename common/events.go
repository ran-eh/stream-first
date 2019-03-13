package common

// This file defines the event types
// used to propagate state through the system.

import (
	"github.com/google/uuid"
	"time"
)

// Static order data, matching the json format in the sample file plus unique order ID.
type Order struct {
	ID        uuid.UUID
	Name      string  `json:"name"`
	Temp      string  `json:"temp"`
	ShelfLife float32 `json:"shelfLife"`
	DecayRate float32 `json:"decayRate"`
}

// pub/sub topics
const (
	NewOrderTopic    = "newOrder"
	ShelvedTopic     = "shelved"
	ReshelvedTopic   = "reshelved"
	PickupTopic      = "pickup"
	ExpiredTopic     = "expired"
	ValueTopic       = "value"
	UserRequestTopic = "keyboard"
	DiagTopic        = "diag"
)

// A mew order arrived
type NewOrderEvent struct {
	Dt    time.Time
	Order Order
}

// A new order was shelved for the first time
type ShelvedEvent struct {
	Dt    time.Time
	Shelf string
	Order Order
}

// An order was moved from the overflow shelf to primary
type ReshelvedEvent struct {
	Dt      time.Time
	OrderID uuid.UUID
}

// WasteEvent fires when an order is declared waste
type WasteEvent struct {
	Dt     time.Time
	Order  Order
	Reason string
}

// An order was picked up
type PickupEvent struct {
	Dt    time.Time
	Order Order
}

// An order expired
type ExpiredEvent struct {
	Dt    time.Time
	Order Order
}

// The order shelf life manager posted an order's value
type ValueEvent struct {
	Dt        time.Time
	Shelf     string
	Value     float32
	NormValue float32
	Order     Order
}

type DiagEvent struct {
	Dt          time.Time
	ServiceName string
	Severity    string
	Message     string
	Error       error
}
