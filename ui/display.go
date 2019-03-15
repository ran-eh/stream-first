package ui

import (
	"fmt"
	"sort"
	"stream-first/event"
	"time"

	"github.com/cskr/pubsub"
	"github.com/google/uuid"

	tm "github.com/buger/goterm"
)

type warehouse struct {
	orders     map[uuid.UUID]*order
	shelves    map[string]*shelf
	wasteCount int
}

type shelf struct {
	OrderSet  map[uuid.UUID]bool
	freeSlots []int
}

type order struct {
	id        uuid.UUID
	name      string
	temp      string
	shelf     string
	value     float32
	normValue float32
	lastMoved time.Time
	slotNo    int
}

var tempVis = map[string]string{"frozen": "[..]", "cold": "[--]", "hot": "[^^]"}

func (o order) String() string {
	return fmt.Sprintf("%v %4.3f : %-6.3f : %v\n", tempVis[o.temp], o.normValue, o.value, o.name)
}

func newWarehouse() *warehouse {
	orders := map[uuid.UUID]*order{}
	shelves := map[string]*shelf{
		"frozen":   &shelf{map[uuid.UUID]bool{}, []int{}},
		"cold":     &shelf{map[uuid.UUID]bool{}, []int{}},
		"hot":      &shelf{map[uuid.UUID]bool{}, []int{}},
		"overflow": &shelf{map[uuid.UUID]bool{}, []int{}},
	}
	oss := &warehouse{orders, shelves, 0}
	return oss
}

func (w *warehouse) SetValue(e *event.ValueEvent) {
	o, ok := w.orders[e.Order.ID]
	if !ok {
		o = &order{id: e.Order.ID, name: e.Order.Name, temp: e.Order.Temp, shelf: e.Shelf}
		w.orders[e.Order.ID] = o
		s := w.shelves[o.shelf]
		s.OrderSet[o.id] = true
	}
	o.value, o.normValue, o.lastMoved = e.Value, e.NormValue, e.LastMoved
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
	tm.Clear()
	// screen.MoveTopLeft()
	//goterm.MoveCursor(1, 1)
	//for _, s := range []string{"frozen", "cold", "hot", "overflow"} {
	//	os := w.getOrders(s)
	//	goterm.Printf("====== %v ======\n", s)
	//	for _, o := range os {
	//		o.render()
	//	}
	//	// goterm.Flush()
	//}
	//goterm.Println("====== WASTE ======")
	//goterm.Println(w.wasteCount)
	//goterm.Flush()

	//table := tm.NewTable(0, 10, 5, ' ', 0)
	//fmt.Fprintf(table, "frozen\tcold\thot\toverflow\n")
	//fmt.Fprintf(table, "%s\t%d\t%d\t%d\n", "All", started, started-finished, finished)
	//tm.Println(totals)
	//tm.Flush()
	shelves := []string{"frozen", "cold", "hot", "overflow"}
	boxes := [4]*tm.Box{}
	width := 50
	for ix, shelf := range shelves {
		orders := w.getOrders(shelf)
		sort.SliceStable(orders, func(i, j int) bool { return orders[i].lastMoved.After(orders[j].lastMoved) })
		boxes[ix] = tm.NewBox(width|tm.PCT, 20, 0)
		_, _ = fmt.Fprintf(boxes[ix], "%v\n", shelf)
		for _, o := range orders {
			_, _ = fmt.Fprintf(boxes[ix], o.String())
		}
		_, _ = tm.Print(tm.MoveTo(boxes[ix].String(), ix*width+10|tm.PCT, 1))
	}
	tm.Flush()

}

func (w warehouse) getOrders(shelf string) (os []order) {
	s := w.shelves[shelf]
	for orderID := range s.OrderSet {
		os = append(os, *w.orders[orderID])
	}
	return
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
				fmt.Printf("could not coerce to event: %v, valueCh, %+v\n", serviceName, msg)
				continue
			}
			w.SetValue(e)
		case msg := <-pickupCh:
			e, ok := msg.(*event.PickupEvent)
			if !ok {
				fmt.Printf("could not coerce to event: display, pickupCh, %+v\n", msg)
				continue
			}
			w.remove(e.Order.ID)
		case msg := <-expiredCh:
			e, ok := msg.(*event.ExpiredEvent)
			if !ok {
				fmt.Printf("could not coerce to event: display, expiredCh, %+v\n", msg)
				continue
			}
			w.remove(e.Order.ID)
		case msg := <-wasteCh:
			// fmt.Print("WSv ")
			_, ok := msg.(*event.WasteEvent)
			if !ok {
				fmt.Printf("could not coerce to event: display, wasteCh, %+v\n", msg)
				continue
			}
			w.waste()
		}
		w.render()
	}
}
