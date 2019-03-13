package main

import (
	"stream-first/display"
	"stream-first/keyboard"
	"stream-first/pickup"
	"stream-first/shelflife"
	"stream-first/shelves"
	input "stream-first/simulatedinputstream"

	"github.com/cskr/pubsub"
)

type orderStates struct {
}

// func displayRun(ps *pubsub.PubSub) {
// 	newOrderCh := ps.Sub(event.EventTypeNewOrder)
// 	shelvedCh := ps.Sub(event.EventTypeShelved)
// 	reshelvedCh := ps.Sub(event.EventTypeReshelved)
// 	wasteCh := ps.Sub(event.EventTypeWaste)
// 	pickupCh := ps.Sub(event.EventTypePickup)
// 	expiredCh := ps.Sub(event.EventTypeExpired)
// 	valueCh := ps.Sub(event.EventTypeValue)

// 	for {
// 		select {
// 		case msg := <-newOrderCh:
// 			e, ok := msg.(*event.NewOrderEvent)
// 			if !ok {
// 				// Error
// 				fmt.Printf("could not coerce to event: display, New, %+v\n", msg)
// 				continue
// 			}

// 			fmt.Printf(">>>>display: %T\n%+v\n", e, e)
// 		case msg := <-shelvedCh:
// 			e, ok := msg.(*event.ShelvedEvent)
// 			if !ok {
// 				// Error
// 				fmt.Printf("could not coerce to event: display, Shelved, %+v\n", msg)
// 				continue
// 			}
// 			fmt.Printf(">>>>display: %T\n%+v\n", e, e)
// 		case msg := <-reshelvedCh:
// 			e, ok := msg.(*event.ReshelvedEvent)
// 			if !ok {
// 				// Error
// 				fmt.Printf("could not coerce to event: display, reshelvedCh, %+v\n", msg)
// 				continue
// 			}
// 			fmt.Printf(">>>>display: %T\n%+v\n", e, e)
// 		case msg := <-wasteCh:
// 			e, ok := msg.(*event.WasteEvent)
// 			if !ok {
// 				// Error
// 				fmt.Printf("could not coerce to event: display, wasteCh, %+v\n", msg)
// 				continue
// 			}
// 			fmt.Printf(">>>>display: %T\n%+v\n", e, e)
// 		case msg := <-pickupCh:
// 			e, ok := msg.(*event.PickupEvent)
// 			if !ok {
// 				// Error
// 				fmt.Printf("could not coerce to event: display, pickupCh, %+v\n", msg)
// 				continue
// 			}
// 			fmt.Printf(">>>>display: %T\n%+v\n", e, e)
// 		case msg := <-expiredCh:
// 			e, ok := msg.(*event.ExpiredEvent)
// 			if !ok {
// 				// Error
// 				fmt.Printf("could not coerce to event: display, expiredCh, %+v\n", msg)
// 				continue
// 			}
// 			fmt.Printf(">>>>display: %T\n%+v\n", e, e)
// 		case msg := <-valueCh:
// 			e, ok := msg.(*event.ValueEvent)
// 			if !ok {
// 				// Error
// 				fmt.Printf("could not coerce to event: display, valueCh, %+v\n", msg)
// 				continue
// 			}
// 			fmt.Printf(">>>>display: %T\n%+v\n", e, e)
// 		}
// 	}
// }

func main() {
	// oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	// if err != nil {
	// 	fmt.Printf("error: %+v\n", err)
	// }
	// fmt.Printf("made it: %+v", oldState)

	// defer terminal.Restore(0, oldState)

	// reader := bufio.NewReader(os.Stdin)

	// for {
	// 	rune, _, err := reader.ReadRune()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	fmt.Printf("Rune: %v\n", rune)
	// 	if rune == 'A' {
	// 		break
	// 	}
	// }

	// tm.Clear() // Clear current screen
	// h := tm.Height()
	// if h <= 0 {

	// 	for {
	// 		// By moving cursor to top-left position we ensure that console output
	// 		// will be overwritten each time, instead of adding new.
	// 		tm.MoveCursor(1, 1)

	// 		tm.Println("Current Time:", time.Now().Format(time.RFC1123))

	// 		tm.Flush() // Call it every time at the end of rendering

	// 		time.Sleep(time.Second)
	// 	}
	// }
	//kbdCh := make(chan rune)

	ps := pubsub.New(1000)

	go keyboard.Run(kbdCh)
	// go displayRun(ps)
	go display.Run(ps)
	go input.Run(ps)
	go shelves.Run(ps)
	go shelflife.Run(ps)
	go pickup.Run(ps)

	// Block forever
	select {}
}
