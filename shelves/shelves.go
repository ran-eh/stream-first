package shelves

import (
	"fmt"
	"github.com/pkg/errors"
	"time"

	"stream-first/event"

	"github.com/cskr/pubsub"
	"github.com/google/uuid"
)

// A non overflow shelf
type primaryShelf struct {
	capacity int
	slots    map[uuid.UUID]bool
}

func newPrimaryShelf(capacity int) *primaryShelf {
	return &primaryShelf{capacity: capacity, slots: make(map[uuid.UUID]bool, capacity)}
}

func (s primaryShelf) has(orderID uuid.UUID) bool {
	return s.slots[orderID]
}

func (s *primaryShelf) store(orderID uuid.UUID) bool {
	if s.slots[orderID] { // Already stored
		return true
	}
	if len(s.slots) >= s.capacity {
		return false
	}
	s.slots[orderID] = true
	return true
}

func (s *primaryShelf) remove(orderID uuid.UUID) bool {
	if !s.slots[orderID] {
		return false
	}
	delete(s.slots, orderID)
	return true
}

// The overflow shelf is modeled by three priority queues, one per temp.  When
// shelf space becomes available, the order with the highest decay factor is moved
// to the main shelf for that temp.

type overflowShelf struct {
	capacity int
	slots    map[string]map[uuid.UUID]float32
}

func newOverflowShelf(capacity int, temps []string) (shelf overflowShelf) {
	slots := map[string]map[uuid.UUID]float32{}
	for _, temp := range temps {
		slots[temp] = map[uuid.UUID]float32{}
	}
	shelf = overflowShelf{capacity: capacity, slots: slots}
	return
}

func (s *overflowShelf) numOrders() int {
	n := 0
	for _, slotSection := range s.slots {
		n += len(slotSection)
	}
	return n
}

func (s *overflowShelf) store(orderID uuid.UUID, temp string, dacayRate float32) (bool, error) {
	if s.numOrders() >= s.capacity {
		return false, nil
	}
	slotSection, ok := s.slots[temp]
	if !ok {
		// Error
		return false, errors.Errorf("Invalid temp: %v", temp)
	}
	_, ok = slotSection[orderID]
	if ok {
		return true, nil
	}
	slotSection[orderID] = dacayRate
	return true, nil
}

func (s *overflowShelf) remove(orderID uuid.UUID, temp string) bool {
	slotSection, ok := s.slots[temp]
	if !ok {
		// Error
		return false
	}
	_, ok = slotSection[orderID]
	if !ok {
		// Error
		return false
	}
	delete(slotSection, orderID)
	return true
}

func (s *overflowShelf) popMax(temp string) (uuid.UUID, bool) {
	found := false
	var maxOrderID uuid.UUID
	maxDecayRate := float32(1)
	slotSection, ok := s.slots[temp]
	if !ok {
		// Error
		return maxOrderID, found
	}
	for orderID, decayRate := range slotSection {
		if decayRate > maxDecayRate {
			maxDecayRate = decayRate
			maxOrderID = orderID
			found = true
		}
	}

	delete(slotSection, maxOrderID)
	return maxOrderID, found
}

type pickupShelves struct {
	shelves  map[string]*primaryShelf
	overflow overflowShelf
	ps       *pubsub.PubSub
}

func newPickupShelves(ps *pubsub.PubSub) *pickupShelves {
	shelves := map[string]*primaryShelf{
		"hot":    newPrimaryShelf(15),
		"cold":   newPrimaryShelf(15),
		"frozen": newPrimaryShelf(15),
	}
	overflow := newOverflowShelf(20, []string{"hot", "cold", "frozen"})
	return &pickupShelves{shelves, overflow, ps}
}

func (pus *pickupShelves) store(order event.Order, temp string, Dt time.Time) bool {
	primaryShelf := pus.shelves[temp]
	if primaryShelf == nil {
		// Error
		fmt.Printf("Invalid temp: %+v\n", temp)
		return false
	}
	stored := primaryShelf.store(order.ID)
	if stored {
		shelvedEvent := &event.ShelvedEvent{Dt: Dt, Order: order, Shelf: temp}
		// fmt.Print("SH^ ")

		pus.ps.Pub(shelvedEvent, event.EventTypeShelved)
		return true
	}
	stored, _ = pus.overflow.store(order.ID, temp, order.DecayRate)
	// fmt.Printf("Stored: %+v, in %+v, %+v\n", stored, "overflow", order.ID)
	if stored {
		shelvedEvent := &event.ShelvedEvent{Dt: Dt, Order: order, Shelf: "overflow"}
		// fmt.Print("SH^ ")
		pus.ps.Pub(shelvedEvent, event.EventTypeShelved)
		return true
	}
	return false
}

func (pus *pickupShelves) remove(orderID uuid.UUID, temp string) bool {
	primaryShelf := pus.shelves[temp]
	if primaryShelf == nil {
		// Error
		fmt.Printf("Invalid temp: %+v\n", temp)
		return false
	}
	if ok := primaryShelf.remove(orderID); ok {
		return true
	}
	if ok := pus.overflow.remove(orderID, temp); ok {
		pus.reshelf(temp)
		return ok
	}
	return false
}

func (pus *pickupShelves) reshelf(temp string) bool {
	primaryShelf := pus.shelves[temp]
	if primaryShelf == nil {
		// Error
		fmt.Printf("Invalid temp: %+v\n", temp)
		return false
	}
	orderID, ok := pus.overflow.popMax(temp)
	if !ok {
		// Error
		return false
	}
	primaryShelf.store(orderID)
	return true
}

// Run runs
func Run(ps *pubsub.PubSub) {
	pickupShelves := newPickupShelves(ps)

	newOrderCh := ps.Sub(event.EventTypeNewOrder)
	pickUpCh := ps.Sub(event.EventTypePickup)
	expiredCh := ps.Sub(event.EventTypeExpired)

	for {
		select {
		case msg := <-newOrderCh:
			// fmt.Print("NW*v ")
			// fmt.Printf("newOrder received: %+v\n", msg)
			newOrderEvent, ok := msg.(*event.NewOrderEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: shelves, new, %+v\n", msg)
				continue
			}
			stored := pickupShelves.store(newOrderEvent.Order, newOrderEvent.Order.Temp, newOrderEvent.Dt)
			if !stored {
				wasteEvent := &event.WasteEvent{Dt: newOrderEvent.Dt, Order: newOrderEvent.Order, Reason: event.WasteReasonNoShelfSpace}
				// fmt.Print("WS^ ")
				ps.Pub(wasteEvent, event.EventTypeWaste)
			}
		case msg := <-pickUpCh:
			// fmt.Print("PU*v ")
			pickupEvent, ok := msg.(*event.PickupEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: shelves, pickup, %+v\n", msg)
				continue
			}
			pickupShelves.remove(pickupEvent.Order.ID, pickupEvent.Order.Temp)
		case msg := <-expiredCh:
			// fmt.Print("EXv ")
			expiredEvent, ok := msg.(*event.ExpiredEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: shelves, expired, %+v\n", msg)
				continue
			}
			pickupShelves.remove(expiredEvent.Order.ID, expiredEvent.Order.Temp)
		}
	}
}
