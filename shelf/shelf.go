package shelf

import (
	"fmt"
	"github.com/pkg/errors"
	"time"

	"stream-first/common"

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

func (s overflowShelf) getSlotSection(temp string) (section map[uuid.UUID]float32, err error) {
	section, ok := s.slots[temp]
	if !ok {
		err = errors.Errorf("invalid temp: %v", temp)
	}
	return
}

func (s *overflowShelf) numOrders() int {
	n := 0
	for _, slotSection := range s.slots {
		n += len(slotSection)
	}
	return n
}

func (s *overflowShelf) store(orderID uuid.UUID, temp string, decayRate float32) (bool, error) {
	if s.numOrders() >= s.capacity {
		return false, nil
	}
	slotSection, err := s.getSlotSection(temp)
	if err != nil {
		return false, err
	}
	_, ok := slotSection[orderID]
	if ok {
		return true, nil
	}
	slotSection[orderID] = decayRate
	return true, nil
}

func (s *overflowShelf) has(orderID uuid.UUID, temp string) (bool, error) {
	slotSection, err := s.getSlotSection(temp)
	if err != nil {
		return false, err
	}
	_, ok := slotSection[orderID]
	return ok, nil
}

func (s *overflowShelf) remove(orderID uuid.UUID, temp string) (bool, error) {
	slotSection, err := s.getSlotSection(temp)
	if err != nil {
		return false, err
	}
	_, ok := slotSection[orderID]
	if !ok {
		// Error
		return false, errors.Errorf("cannot remove order %v from section %v: not in section", orderID, temp)
	}
	delete(slotSection, orderID)
	return true, nil
}

func (s *overflowShelf) popMax(temp string) (maxOrderID uuid.UUID, found bool, err error) {
	slotSection, err0 := s.getSlotSection(temp)
	if err0 != nil {
		err = err0
		return
	}

	maxDecayRate := float32(-1)

	if len(slotSection) == 0 {
		return
	}

	for orderID, decayRate := range slotSection {
		if decayRate > maxDecayRate {
			maxDecayRate = decayRate
			maxOrderID = orderID
			found = true
		}
	}

	delete(slotSection, maxOrderID)
	return
}

type warehouse struct {
	shelves  map[string]*primaryShelf
	overflow overflowShelf
	ps       common.PubSubInterface
}

func newWarehouse(ps common.PubSubInterface, primaryCapacity int, overflowCapacity int) *warehouse {
	shelves := map[string]*primaryShelf{
		"hot":    newPrimaryShelf(primaryCapacity),
		"cold":   newPrimaryShelf(primaryCapacity),
		"frozen": newPrimaryShelf(primaryCapacity),
	}
	overflow := newOverflowShelf(overflowCapacity, []string{"hot", "cold", "frozen"})
	return &warehouse{shelves, overflow, ps}
}

func (w *warehouse) store(order common.Order, temp string, Dt time.Time) (stored bool, err error) {
	primaryShelf := w.shelves[temp]
	if primaryShelf == nil {
		err = errors.Errorf("Invalid temp: %+v", temp)
		return
	}
	stored = primaryShelf.store(order.ID)
	if stored {
		shelvedEvent := &common.ShelvedEvent{Dt: Dt, Order: order, Shelf: temp}

		w.ps.Pub(shelvedEvent, common.EventTypeShelved)
		return
	}
	stored, err = w.overflow.store(order.ID, temp, order.DecayRate)
	if stored {
		shelvedEvent := &common.ShelvedEvent{Dt: Dt, Order: order, Shelf: "overflow"}
		// fmt.Print("SH^ ")
		w.ps.Pub(shelvedEvent, common.EventTypeShelved)
		return
	}
	return
}

func (w *warehouse) has(orderID uuid.UUID, temp string) (shelf string, found bool, err error) {
	primaryShelf := w.shelves[temp]
	if primaryShelf == nil {
		err = errors.Errorf("Invalid temp: %+v", temp)
		return
	}
	found = primaryShelf.has(orderID)
	if found {
		shelf = temp
		return
	}
	found, err = w.overflow.has(orderID, temp)
	if found {
		shelf = "overflow"
	}
	return
}

func (w *warehouse) remove(orderID uuid.UUID, temp string, dt time.Time) (done bool, err error) {
	primaryShelf := w.shelves[temp]
	if primaryShelf == nil {
		err = errors.Errorf("Invalid temp: %+v", temp)
		return
	}
	done = primaryShelf.remove(orderID)
	if done {
		_, err = w.reshelf(temp, dt)
		return
	}
	done, err = w.overflow.remove(orderID, temp)
	return
}

func (w *warehouse) reshelf(temp string, dt time.Time) (found bool, err error) {
	primaryShelf := w.shelves[temp]
	if primaryShelf == nil {
		err = errors.Errorf("Invalid temp: %+v", temp)
		return
	}
	var orderID uuid.UUID
	orderID, found, err = w.overflow.popMax(temp)
	if err != nil || !found {
		// Error
		return
	}
	primaryShelf.store(orderID)
	reshelvedEvent := &common.ReshelvedEvent{dt, orderID}
	w.ps.Pub(reshelvedEvent, common.EventTypeReshelved)
	return
}

// Run runs
func Run(ps *pubsub.PubSub) {
	w := newWarehouse(ps, 15, 20)

	newOrderCh := ps.Sub(common.EventTypeNewOrder)
	pickUpCh := ps.Sub(common.EventTypePickup)
	expiredCh := ps.Sub(common.EventTypeExpired)

	for {
		select {
		case msg := <-newOrderCh:
			// fmt.Print("NW*v ")
			// fmt.Printf("newOrder received: %+v\n", msg)
			newOrderEvent, ok := msg.(*common.NewOrderEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: shelves, new, %+v\n", msg)
				continue
			}
			stored, _ := w.store(newOrderEvent.Order, newOrderEvent.Order.Temp, newOrderEvent.Dt)
			if !stored {
				wasteEvent := &common.WasteEvent{Dt: newOrderEvent.Dt, Order: newOrderEvent.Order, Reason: common.WasteReasonNoShelfSpace}
				// fmt.Print("WS^ ")
				ps.Pub(wasteEvent, common.EventTypeWaste)
			}
		case msg := <-pickUpCh:
			pickupEvent, ok := msg.(*common.PickupEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: shelves, pickup, %+v\n", msg)
				continue
			}
			w.remove(pickupEvent.Order.ID, pickupEvent.Order.Temp, pickupEvent.Dt)
		case msg := <-expiredCh:
			// fmt.Print("EXv ")
			expiredEvent, ok := msg.(*common.ExpiredEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: shelves, expired, %+v\n", msg)
				continue
			}
			w.remove(expiredEvent.Order.ID, expiredEvent.Order.Temp, expiredEvent.Dt)
		}
	}
}
