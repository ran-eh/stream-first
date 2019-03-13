package io

import (
	"fmt"
	"stream-first/event"

	"github.com/cskr/pubsub"
	"github.com/google/uuid"

	// "github.com/inancgumus/screen"
	"github.com/buger/goterm"
)

type warehouse struct {
	orders     map[uuid.UUID]*order
	shelves    map[string]*shelf
	wasteCount int
}

type shelf struct {
	OrderSet map[uuid.UUID]bool
}

type order struct {
	id        uuid.UUID
	name      string
	temp      string
	shelf     string
	value     float32
	normValue float32
}

var tempVis = map[string]string{"frozen": "[..]", "cold": "[--]", "hot": "[^^]"}

func (o order) render() {
	goterm.Printf("%v%v : %v : %v\n", tempVis[o.temp], o.normValue, o.value, o.name)
}

func newWarehouse() *warehouse {
	orders := map[uuid.UUID]*order{}
	shelves := map[string]*shelf{
		"frozen":   &shelf{map[uuid.UUID]bool{}},
		"cold":     &shelf{map[uuid.UUID]bool{}},
		"hot":      &shelf{map[uuid.UUID]bool{}},
		"overflow": &shelf{map[uuid.UUID]bool{}},
	}
	oss := &warehouse{orders, shelves, 0}
	return oss
}

func (w *warehouse) SetValue(e *event.ValueEvent) {
	o, ok := w.orders[e.Order.ID]
	if !ok {
		o = &order{id: e.Order.ID, name: e.Order.Name, temp: e.Order.Temp, shelf: e.Shelf}
		w.orders[e.Order.ID] = o
		w.shelves[o.shelf].OrderSet[o.id] = true
	}
	o.value, o.normValue = e.Value, e.NormValue
	if o.shelf != e.Shelf {
		delete(w.shelves[o.shelf].OrderSet, o.id)
		o.shelf = e.Shelf
		w.shelves[o.shelf].OrderSet[o.id] = true
	}
}

func (w *warehouse) remove(orderID uuid.UUID) {
	o := w.orders[orderID]
	delete(w.shelves[o.shelf].OrderSet, o.id)
}

func (w *warehouse) waste() {
	w.wasteCount++
}

func (w warehouse) render() {
	goterm.Clear()
	// screen.MoveTopLeft()
	goterm.MoveCursor(1, 1)
	for _, s := range []string{"frozen", "cold", "hot", "overflow"} {
		os := w.getOrders(s)
		goterm.Printf("====== %v ======\n", s)
		for _, o := range os {
			o.render()
		}
		// goterm.Flush()
	}
	goterm.Println("====== WASTE ======")
	goterm.Println(w.wasteCount)
	goterm.Flush()
}

func (w warehouse) getOrders(shelf string) []order {
	os := []order{}
	s := w.shelves[shelf]
	for orderID := range s.OrderSet {
		os = append(os, *w.orders[orderID])
	}
	return os
}

const serviceName = "display"

// Run runs
func Run(ps *pubsub.PubSub) {

	w := newWarehouse()

	valueCh := ps.Sub(event.EventTypeValue)
	pickupCh := ps.Sub(event.EventTypePickup)
	expiredCh := ps.Sub(event.EventTypeExpired)
	wasteCh := ps.Sub(event.EventTypeWaste)

	for {
		select {
		case msg := <-valueCh:
			// fmt.Print("VLv ")
			e, ok := msg.(*event.ValueEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: %v, valueCh, %+v\n", serviceName, msg)
				continue
			}
			// fmt.Printf(">>>>%v: %T\n%+v\n", serviceName, e, e)
			w.SetValue(e)
		case msg := <-pickupCh:
			// fmt.Print("PUv ")
			e, ok := msg.(*event.PickupEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: display, pickupCh, %+v\n", msg)
				continue
			}
			// fmt.Printf(">>>>display: %T\n%+v\n", e, e)
			w.remove(e.Order.ID)
		case msg := <-expiredCh:
			// fmt.Print("EXv ")
			e, ok := msg.(*event.ExpiredEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: display, expiredCh, %+v\n", msg)
				continue
			}
			// fmt.Printf(">>>>display: %T\n%+v\n", e, e)
			w.remove(e.Order.ID)
		case msg := <-wasteCh:
			// fmt.Print("WSv ")
			_, ok := msg.(*event.WasteEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: display, wasteCh, %+v\n", msg)
				continue
			}
			// fmt.Printf(">>>>display: %T\n%+v\n", e, e)
			w.waste()
		}
		w.render()
	}
}
