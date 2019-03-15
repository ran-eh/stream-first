package shelflife

import (
	"fmt"
	"stream-first/event"
	"time"

	"github.com/cskr/pubsub"
	"github.com/google/uuid"
)

type orderState struct {
	order             *event.Order
	shelf             string
	startTimePrimary  time.Time
	startTimeOverflow time.Time
}

func (s orderState) value(now time.Time) float32 {
	durationAtOverflow := time.Duration(0)
	durationAtPrimary := time.Duration(0)
	if !s.startTimeOverflow.IsZero() {
		if s.startTimePrimary.IsZero() {
			durationAtOverflow = now.Sub(s.startTimeOverflow)
		} else {
			durationAtOverflow = s.startTimePrimary.Sub(s.startTimeOverflow)
		}
	}
	if !s.startTimePrimary.IsZero() {
		durationAtPrimary = now.Sub(s.startTimePrimary)
	}
	age := durationAtOverflow + durationAtOverflow
	shelfLife := float64(s.order.ShelfLife)
	decayRate := float64(s.order.DecayRate)
	value := shelfLife - age.Seconds() - decayRate*durationAtPrimary.Seconds() - decayRate*2*durationAtOverflow.Seconds()
	if value < 0 {
		value = 0
	}
	return float32(value)
}

func (s orderState) lastMoved(t1, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}

// Run runs
func Run(ps *pubsub.PubSub) {
	orderStates := map[uuid.UUID]*orderState{}

	shelvedCh := ps.Sub(event.EventTypeShelved)
	reshelvedCh := ps.Sub(event.EventTypeReshelved)
	pickupCh := ps.Sub(event.EventTypePickup)

	fmt.Println("Starting shelflife loop")

	for {
		select {
		case msg := <-shelvedCh:
			// fmt.Print("SHv ")
			shelvedEvent, ok := msg.(*event.ShelvedEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: shelfline, Shelved, %+v\n", msg)
			}
			state, ok := orderStates[shelvedEvent.Order.ID]
			if !ok {
				state = &orderState{order: &shelvedEvent.Order, shelf: shelvedEvent.Shelf}
				orderStates[shelvedEvent.Order.ID] = state
			}
			if shelvedEvent.Shelf == "overflow" {
				state.startTimeOverflow = shelvedEvent.Dt
			} else {
				state.startTimePrimary = shelvedEvent.Dt
			}
		case msg := <-reshelvedCh:
			// fmt.Print("RSv ")
			reshelvedEvent, ok := msg.(*event.ReshelvedEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: shelfline, Reshelved, %+v\n", msg)
			}
			state, ok := orderStates[reshelvedEvent.OrderID]
			if !ok {
				// error
				continue
			}
			state.startTimePrimary = reshelvedEvent.Dt
		case msg := <-pickupCh:
			// fmt.Print("PUv ")
			pickupEvent, ok := msg.(*event.PickupEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: shelfline, Pickup, %+v\n", msg)
			}
			delete(orderStates, pickupEvent.Order.ID)
		}

		now := time.Now()
		for orderID, state := range orderStates {
			value := state.value(now)
			if value <= 0 {
				delete(orderStates, orderID)
				ps.Pub(&event.ExpiredEvent{Dt: now, Order: *state.order}, event.EventTypeExpired)
			} else {
				value := state.value(now)
				normValue := value / state.order.ShelfLife

				lastMoved := state.lastMoved(state.startTimePrimary, state.startTimeOverflow)

				ps.Pub(&event.ValueEvent{
					Dt:        now,
					Order:     *state.order,
					Shelf:     state.shelf,
					Value:     value,
					NormValue: normValue,
					LastMoved: lastMoved,
				},
					event.EventTypeValue)
			}
		}
	}
}
