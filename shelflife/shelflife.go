package shelflife

import (
	"fmt"
	"github.com/cskr/pubsub"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"stream-first/common"
	"time"
)

// The shelflife package calculates and publishes the value of shelved orders.  The publishing of updates values
// happens when an order is shelved, re-shelved, picked up or becomes expired.  A keep-alive timer triggers publishing
// if none of the other events occurred in while.  This will happen when the new order arrival and pickups are both
// paused for testing.

const (
	serviceName             = "ShelfLife"
	keepAliveTriggerSeconds = 1.0
)

// OrderState holds the information needed to calculate the order value.
type OrderState struct {
	Order *common.Order
	Shelf string
	// Order was placed on the primary shelf at....
	// It will have the zero value if it was never on that shelf.
	TimePlacedOnPrimaryShelf time.Time
	// Order was placed on the overflow shelf at....
	// It will have the zero value if it was never on that shelf.
	TimePlacedOnOverflowShelf time.Time
}

// Maps order IDs to order states.
var OrderStates = map[uuid.UUID]*OrderState{}

func ResetStates() {
	OrderStates = map[uuid.UUID]*OrderState{}
}

func (s OrderState) Value(now time.Time) (value float32, err error) {
	if s.TimePlacedOnOverflowShelf.IsZero() && s.TimePlacedOnPrimaryShelf.IsZero() {
		err = errors.Errorf("impossible: order was never shelved on neither primary nor overflow: %v", s.Order.ID)
		return
	}
	if !s.TimePlacedOnOverflowShelf.IsZero() && !s.TimePlacedOnPrimaryShelf.IsZero() &&
		s.TimePlacedOnOverflowShelf.After(s.TimePlacedOnPrimaryShelf) {
		err = errors.Errorf("reshelved from primary to overflow, which should never happen, for Order %v", s.Order.ID)
		return
	}

	// Calculate how long the orders spent on the primary shelf, and how long in overflow.
	var durationAtOverflow, durationAtPrimary time.Duration

	// If the order spent time on the overflow shelf
	if !s.TimePlacedOnOverflowShelf.IsZero() {
		if s.TimePlacedOnPrimaryShelf.IsZero() { // No time spent on primary shelf
			durationAtOverflow = now.Sub(s.TimePlacedOnOverflowShelf)
		} else { // Order was reshelved to primary
			durationAtOverflow = s.TimePlacedOnPrimaryShelf.Sub(s.TimePlacedOnOverflowShelf)
		}
	}

	// If the order spent time on the primary shelf
	if !s.TimePlacedOnPrimaryShelf.IsZero() {
		durationAtPrimary = now.Sub(s.TimePlacedOnPrimaryShelf)
	}
	age := durationAtPrimary + durationAtOverflow
	decayWhileOnPrimary := float64(s.Order.DecayRate) * durationAtPrimary.Seconds()
	// While on on overflow, orders decay at twice the normal rate.
	decayWhileOnOverflow := float64(s.Order.DecayRate) * 2 * durationAtOverflow.Seconds()
	value = float32(float64(s.Order.ShelfLife) - age.Seconds() - decayWhileOnPrimary - decayWhileOnOverflow)

	// Can't be more expired than expired.
	if value < 0 {
		value = 0
	}
	return
}

func Run(ps *pubsub.PubSub) {
	shelvedCh := ps.Sub(common.ShelvedTopic)
	reshelvedCh := ps.Sub(common.ReshelvedTopic)
	pickupCh := ps.Sub(common.PickupTopic)

	// Allow time for other components to subscribe before starting to publish.
	time.Sleep(common.Seconds(common.SchedulerDelay))

	common.Diag(ps, serviceName, common.Info, "Service started.", nil)

	// TODO: document the fact that for the ui, I chose once a second

	Run0(ps, shelvedCh, reshelvedCh, pickupCh, nil)
}

func Run0(ps common.PubsubInterface,
	shelvedCh chan interface{}, reshelvedCh chan interface{}, pickupCh chan interface{}, stopCh chan bool) {

	keepAliveCh := time.NewTimer(common.Seconds(keepAliveTriggerSeconds))

	for {
		select {
		case msg := <-shelvedCh:
			e, ok := msg.(*common.ShelvedEvent)
			if !ok {
				common.Diag(ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			state, ok := OrderStates[e.Order.ID]
			if !ok {
				state = &OrderState{Order: &e.Order, Shelf: e.Shelf}
				OrderStates[e.Order.ID] = state
			}
			if e.Shelf == "overflow" {
				state.TimePlacedOnOverflowShelf = e.Dt
			} else {
				state.TimePlacedOnPrimaryShelf = e.Dt
			}
		case msg := <-reshelvedCh:
			e, ok := msg.(*common.ReshelvedEvent)
			if !ok {
				common.Diag(ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			// Order may have been picked up
			state, ok := OrderStates[e.OrderID]
			if !ok {
				common.Diag(ps, serviceName, common.Warning, fmt.Sprintf("Reshelf failed, order not found: %v", e.OrderID), nil)
				continue
			}
			state.Shelf = state.Order.Temp
			state.TimePlacedOnPrimaryShelf = e.Dt
		case msg := <-pickupCh:
			e, ok := msg.(*common.PickupEvent)
			if !ok {
				common.Diag(ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			delete(OrderStates, e.Order.ID)
		case <-stopCh:
			break
		case <-keepAliveCh.C: // keep alive when other events are not coming
		}

		now := time.Now()
		for orderID, state := range OrderStates {
			value, _ := state.Value(now)
			if value <= 0 {
				delete(OrderStates, orderID)
				ps.Pub(&common.ExpiredEvent{Dt: now, Order: *state.Order}, common.ExpiredTopic)
				common.Diag(ps, serviceName, common.Warning, fmt.Sprintf("Waste - order expired: %+v", *state.Order), nil)
			} else {
				normValue := value / state.Order.ShelfLife

				ps.Pub(&common.ValueEvent{
					Dt:        now,
					Order:     *state.Order,
					Shelf:     state.Shelf,
					Value:     value,
					NormValue: normValue,
				},
					common.ValueTopic)
			}
		}

		// Reset keep-alive timer

		// The golang docs state that the timer needs to be drained before reset if it did not yet fire -
		// see https://golang.org/pkg/time/#Timer.Reset .  However, this seems not to be the case.  Reset()
		// appears to drain the channel, so using their example code causes deadlock.  Resetting without draining
		// appears to work fine - the timer does not fire extraneously after Reset.

		keepAliveCh.Reset(common.Seconds(keepAliveTriggerSeconds))
	}
}
