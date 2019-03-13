package screen

import (
	"fmt"
	"runtime"
	"stream-first/common"
	"stream-first/ui/userrequests"
	"time"

	"github.com/cskr/pubsub"
	"github.com/google/uuid"

	tm "github.com/buger/goterm"
)

// This package displays the orderState placement on shelves.  It also shoes diagnostics messages as they occur, and the
// paused/resumed state services that can be paused.

const (
	serviceName = "UI/Screen"
)

// Shelf state holds the orders that need to be displayed.  An order has a fixed
// display position on the screen while it's on the shelf, to prevent orders
// from bouncing around the screen over time.
type ShelfState struct {
	ps                       common.PubsubInterface
	OrderIDToDisplayPosition map[uuid.UUID]*int
	DisplayPositionToOrderID []*uuid.UUID
}

func NewShelfState(ps common.PubsubInterface) *ShelfState {
	return &ShelfState{OrderIDToDisplayPosition: map[uuid.UUID]*int{}, DisplayPositionToOrderID: []*uuid.UUID{}, ps: ps}
}

func (s *ShelfState) allocateEmptyDisplayPosition() (displayPosition int) {
	for position, orderID := range s.DisplayPositionToOrderID {
		if orderID == nil {
			return position
		}
	}
	// None found, create a new one
	newPosition := len(s.DisplayPositionToOrderID)
	s.DisplayPositionToOrderID = append(s.DisplayPositionToOrderID, nil)
	return newPosition
}

func (s *ShelfState) Add(orderID uuid.UUID) int {
	displayPosition := s.allocateEmptyDisplayPosition()
	s.DisplayPositionToOrderID[displayPosition] = &orderID
	s.OrderIDToDisplayPosition[orderID] = &displayPosition
	return displayPosition
}

func (s *ShelfState) Remove(orderID uuid.UUID) {
	displayPosition, found := s.OrderIDToDisplayPosition[orderID]
	if !found {
		return
	}
	delete(s.OrderIDToDisplayPosition, orderID)
	s.DisplayPositionToOrderID[*displayPosition] = nil
}

// Holds the information required to render an order.
type orderState struct {
	id        uuid.UUID
	name      string
	temp      string
	shelf     string
	value     float32
	normValue float32
}

func (s orderState) String() string {
	var tempVis = map[string]string{"frozen": "[..]", "cold": "[--]", "hot": "[^^]"}
	return fmt.Sprintf("%v %4.3f : %-6.3f : %v\n", tempVis[s.temp], s.normValue, s.value, s.name)
}

// Holds the data required to render the screen
type state struct {
	orders  map[uuid.UUID]*orderState
	shelves map[string]*ShelfState
	ps      *pubsub.PubSub
	// Diagnostic messages to be displayed.
	diags []common.DiagEvent
}

func newDisplayState(ps *pubsub.PubSub) *state {
	orders := map[uuid.UUID]*orderState{}
	shelves := map[string]*ShelfState{
		"frozen":   NewShelfState(ps),
		"cold":     NewShelfState(ps),
		"hot":      NewShelfState(ps),
		"overflow": NewShelfState(ps),
	}
	return &state{orders: orders, shelves: shelves, ps: ps}
}

func (s *state) update(e *common.ValueEvent) {
	o, ok := s.orders[e.Order.ID]
	if !ok {
		o = &orderState{id: e.Order.ID, name: e.Order.Name, temp: e.Order.Temp, shelf: e.Shelf}
		s.orders[e.Order.ID] = o
		s := s.shelves[o.shelf]
		s.Add(e.Order.ID)
	}
	o.value, o.normValue = e.Value, e.NormValue
	if o.shelf != e.Shelf {
		s.shelves[o.shelf].Remove(o.id)
		s.shelves[e.Shelf].Add(o.id)
		o.shelf = e.Shelf
	}
}

func (s *state) remove(orderID uuid.UUID) {
	o, orderFound := s.orders[orderID]
	if !orderFound {
		common.Diag(s.ps, serviceName, common.Warning, fmt.Sprintf("w.remove: orderState not found: %+v", orderID.String()), nil)
		return
	}
	shelf, shelfFound := s.shelves[o.shelf]
	if !shelfFound {
		common.Diag(s.ps, serviceName, common.Warning, fmt.Sprintf("w.remove: shelf not found: %+v", o.shelf), nil)
	}
	shelf.Remove(o.id)
}

// Render the screen
func (s state) render() {
	var screenWidth int
	// goterm does not properly return screen width on windows.
	// hard code to 160 characters in that case.
	if runtime.GOOS == "windows" {
		screenWidth = 160
	} else {
		screenWidth = tm.Width()
	}
	diagBoxHeight := 23

	tm.Clear()

	// One display box per shelf.
	boxWidth := screenWidth / 4
	boxes := [4]*tm.Box{}

	for column, shelfName := range []string{"frozen", "cold", "hot", "overflow"} {
		var capacity int
		if shelfName == "overflow" {
			capacity = 20
		} else {
			capacity = 15
		}

		shelf := s.shelves[shelfName]

		boxes[column] = tm.NewBox(boxWidth, capacity+3, 0)
		box := boxes[column]

		// Show shelf name at top of the box
		_, _ = fmt.Fprintf(box, "%v\n", shelfName)

		for _, orderID := range shelf.DisplayPositionToOrderID {
			if orderID != nil {
				_, _ = fmt.Fprintf(box, s.orders[*orderID].String())
			} else { // Display position vacant
				_, _ = fmt.Fprintf(box, "\n")
			}
		}
		// Place the box in columnar fashion.
		_, _ = tm.Print(tm.MoveTo(box.String(), column*boxWidth+1, 1))
	}
	// Render the status line below the shelf boxes
	tm.MoveCursor(1, 19)
	_, _ = tm.Printf("%v\r\n", statusLine())

	// Render diagnostics box below the status line
	diagBox := tm.NewBox(3*boxWidth, diagBoxHeight, 0)
	_, _ = fmt.Fprintf(diagBox, "%v\n", "Diagnostics")
	for _, diag := range s.diags {
		_, _ = fmt.Fprintf(diagBox, "%v\n", diag.String())
	}
	_, _ = tm.Print(tm.MoveTo(diagBox.String(), 1, 20))

	// Update the screen
	tm.Flush()
}

func statusLine() (line string) {
	if userrequests.RequestedState.PickUpPaused {
		line += "[P] Toggle Pickup (Paused)  | "
	} else {
		line += "[P] Toggle Pickup (Running) | "
	}
	if userrequests.RequestedState.IncomingOrdersPaused {
		line += "[I] Toggle Incoming Stream (Paused)  | "
	} else {
		line += "[I] Toggle Incoming Stream (Running) | "
	}
	line += "[Q] Quit"
	return
}

func Run(ps *pubsub.PubSub) {
	go userrequests.Run(ps)

	valueCh := ps.Sub(common.ValueTopic)
	pickupCh := ps.Sub(common.PickupTopic)
	expiredCh := ps.Sub(common.ExpiredTopic)
	userRequestCh := ps.Sub(common.UserRequestTopic)
	diagCh := ps.Sub(common.DiagTopic)

	// Allow time for other components to subscribe
	time.Sleep(common.Seconds(common.SchedulerDelay))

	common.Diag(ps, serviceName, common.Info, "Service started.", nil)

	// The spec called for updating the screen every time an order is added and moved, but that causes
	// overloading the display.  Instead, the screen is refreshed once a second.
	tickCh := time.Tick(common.Seconds(1))
	s := newDisplayState(ps)
	for {
		select {
		case msg := <-valueCh:
			e, ok := msg.(*common.ValueEvent)
			if !ok {
				common.Diag(s.ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			s.update(e)
		case msg := <-pickupCh:
			e, ok := msg.(*common.PickupEvent)
			if !ok {
				common.Diag(s.ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			// May have expired
			if s.orders[e.Order.ID] != nil {
				s.remove(e.Order.ID)
			}
		case msg := <-expiredCh:
			e, ok := msg.(*common.ExpiredEvent)
			if !ok {
				common.Diag(s.ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			// May have been picked up
			if s.orders[e.Order.ID] != nil {
				s.remove(e.Order.ID)
			}
		case msg := <-diagCh:
			e, ok := msg.(*common.DiagEvent)
			if !ok {
				common.Diag(s.ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			s.diags = append(s.diags, *e)
			if len(s.diags) > 20 {
				// Drop least recent diagnostic message.
				s.diags = s.diags[1:]
			}
		case <-tickCh:
			s.render()
		case <-userRequestCh:
			// Make the screen responsive to keyboard events
			s.render()
		}
	}
}
