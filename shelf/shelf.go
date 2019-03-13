package shelf

import (
	"fmt"
	"github.com/pkg/errors"
	"time"

	"stream-first/common"

	"github.com/cskr/pubsub"
	"github.com/google/uuid"
)

const serviceName = "Shelf"

// A non overflow shelf
type primaryShelf struct {
	capacity int
	// The set of order IDs for orders on the shelf
	orders map[uuid.UUID]bool
}

func NewPrimaryShelf(capacity int) *primaryShelf {
	return &primaryShelf{capacity: capacity, orders: make(map[uuid.UUID]bool, capacity)}
}

func (s primaryShelf) Has(orderID uuid.UUID) bool {
	return s.orders[orderID]
}

func (s *primaryShelf) Store(orderID uuid.UUID) bool {
	if s.orders[orderID] { // Already stored
		return true
	}
	if len(s.orders) >= s.capacity {
		return false
	}
	s.orders[orderID] = true
	return true
}

func (s *primaryShelf) Remove(orderID uuid.UUID) bool {
	if !s.orders[orderID] {
		return false
	}
	delete(s.orders, orderID)
	return true
}

// The overflow shelf stores orders of all temperatures. To provide for primary shelf space becoming
// available, it allows removing the highest decay rate item for a given temperature.
type OverflowShelf struct {
	Capacity int
	// for each temperature, map order ids to decay rate.
	Orders map[string]map[uuid.UUID]float32
}

func NewOverflowShelf(capacity int, temps []string) (shelf OverflowShelf) {
	orders := map[string]map[uuid.UUID]float32{}
	for _, temp := range temps {
		orders[temp] = map[uuid.UUID]float32{}
	}
	shelf = OverflowShelf{Capacity: capacity, Orders: orders}
	return
}

func (s OverflowShelf) getOrders(temp string) (orders map[uuid.UUID]float32, err error) {
	orders, ok := s.Orders[temp]
	if !ok {
		err = errors.Errorf("invalid temp: %v", temp)
	}
	return
}

func (s *OverflowShelf) NumOrders() int {
	n := 0
	for _, slotSection := range s.Orders {
		n += len(slotSection)
	}
	return n
}

func (s *OverflowShelf) Store(orderID uuid.UUID, temp string, decayRate float32) (stored bool, err error) {
	// If shelf is at capacity, return false
	if s.NumOrders() >= s.Capacity {
		return
	}
	orders, err := s.getOrders(temp)
	if err != nil {
		return
	}

	_, stored = orders[orderID]
	if stored {
		// Stored already, return false
		return
	}
	orders[orderID] = decayRate
	return true, nil
}

func (s *OverflowShelf) Has(orderID uuid.UUID, temp string) (has bool, err error) {
	orders, err := s.getOrders(temp)
	if err != nil {
		return
	}
	_, has = orders[orderID]
	return
}

func (s *OverflowShelf) Remove(orderID uuid.UUID, temp string) (found bool, err error) {
	orders, err := s.getOrders(temp)
	if err != nil {
		return
	}
	_, found = orders[orderID]
	if !found {
		err = errors.Errorf("cannot Remove order %v/%v: not found", orderID, temp)
		return
	}
	delete(orders, orderID)
	return
}

// If shelf has orders for the given temp, return the one with highest decay rate and return it
func (s *OverflowShelf) PopMax(temp string) (maxOrderID uuid.UUID, found bool, err error) {
	var orders map[uuid.UUID]float32
	orders, err = s.getOrders(temp)
	if err != nil {
		return
	}

	if len(orders) == 0 {
		// no orders for temp, return not found.
		return
	}

	maxDecayRate := float32(-1)

	// O(n) scan is quick for small sets.  When size gets bigger,
	// use priority queues instead of slices to Store the orders.
	for orderID, decayRate := range orders {
		if decayRate > maxDecayRate {
			maxDecayRate = decayRate
			maxOrderID = orderID
			found = true
		}
	}

	delete(orders, maxOrderID)
	return
}

type Manager struct {
	shelves  map[string]*primaryShelf
	overflow OverflowShelf
	ps       common.PubsubInterface
}

// TODO: eliminate use of "warehouse" and w.  use m instead
func NewManager(ps common.PubsubInterface, primaryCapacity int, overflowCapacity int) *Manager {
	shelves := map[string]*primaryShelf{
		"hot":    NewPrimaryShelf(primaryCapacity),
		"cold":   NewPrimaryShelf(primaryCapacity),
		"frozen": NewPrimaryShelf(primaryCapacity),
	}
	overflow := NewOverflowShelf(overflowCapacity, []string{"hot", "cold", "frozen"})
	return &Manager{shelves, overflow, ps}
}

func (w *Manager) Store(order common.Order, temp string, Dt time.Time) (stored bool, err error) {
	primaryShelf := w.shelves[temp]
	if primaryShelf == nil {
		err = errors.Errorf("Invalid temp: %+v", temp)
		return
	}
	stored = primaryShelf.Store(order.ID)
	if stored {
		// Successfully stored on primary shelf.
		shelvedEvent := &common.ShelvedEvent{Dt: Dt, Order: order, Shelf: temp}

		w.ps.Pub(shelvedEvent, common.ShelvedTopic)
		return
	}
	stored, err = w.overflow.Store(order.ID, temp, order.DecayRate)
	if stored {
		// Successfully stored on overflow shelf.
		shelvedEvent := &common.ShelvedEvent{Dt: Dt, Order: order, Shelf: "overflow"}
		w.ps.Pub(shelvedEvent, common.ShelvedTopic)
		return
	}
	// Storage failed, return false.
	return
}

func (w *Manager) Has(orderID uuid.UUID, temp string) (shelf string, found bool, err error) {
	primaryShelf := w.shelves[temp]
	if primaryShelf == nil {
		err = errors.Errorf("Invalid temp: %+v", temp)
		return
	}
	found = primaryShelf.Has(orderID)
	if found {
		shelf = temp
		return
	}
	found, err = w.overflow.Has(orderID, temp)
	if found {
		shelf = "overflow"
	}
	return
}

func (w *Manager) Remove(orderID uuid.UUID, temp string, dt time.Time) (done bool, err error) {
	primaryShelf := w.shelves[temp]
	if primaryShelf == nil {
		err = errors.Errorf("Invalid temp: %+v", temp)
		return
	}
	done = primaryShelf.Remove(orderID)
	if done {
		_, err = w.reshelf(temp, dt)
		return
	}
	done, err = w.overflow.Remove(orderID, temp)
	return
}

func (w *Manager) reshelf(temp string, dt time.Time) (found bool, err error) {
	primaryShelf := w.shelves[temp]
	if primaryShelf == nil {
		err = errors.Errorf("Invalid temp: %+v", temp)
		return
	}
	var orderID uuid.UUID
	orderID, found, err = w.overflow.PopMax(temp)
	if err != nil || !found {
		// Error
		return
	}
	primaryShelf.Store(orderID)
	reshelvedEvent := &common.ReshelvedEvent{Dt: dt, OrderID: orderID}
	w.ps.Pub(reshelvedEvent, common.ReshelvedTopic)
	return
}

func Run(ps *pubsub.PubSub) {
	newOrderCh := ps.Sub(common.NewOrderTopic)
	pickUpCh := ps.Sub(common.PickupTopic)
	expiredCh := ps.Sub(common.ExpiredTopic)

	// Allow time for other components to subscribe before starting to publish.
	time.Sleep(common.Seconds(common.SchedulerDelay))

	common.Diag(ps, serviceName, common.Info, "Service started.", nil)

	m := NewManager(ps, 15, 20)
	for {
		select {
		case msg := <-newOrderCh:
			e, ok := msg.(*common.NewOrderEvent)
			if !ok {
				common.Diag(ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			stored, err := m.Store(e.Order, e.Order.Temp, e.Dt)
			if err != nil {
				common.Diag(ps, serviceName, common.Error, "", err)
			}
			if !stored {
				common.Diag(ps, serviceName, common.Error, fmt.Sprintf("Waste - shelves full: %+v", e.Order), nil)
			}
		case msg := <-pickUpCh:
			e, ok := msg.(*common.PickupEvent)
			if !ok {
				common.Diag(ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			_, _ = m.Remove(e.Order.ID, e.Order.Temp, e.Dt)
		case msg := <-expiredCh:
			e, ok := msg.(*common.ExpiredEvent)
			if !ok {
				common.Diag(ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			_, _ = m.Remove(e.Order.ID, e.Order.Temp, e.Dt)
		}
	}
}
