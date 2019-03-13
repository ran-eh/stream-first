package pickup

import (
	"fmt"
	"stream-first/event"
	"time"

	"github.com/cskr/pubsub"
	"gonum.org/v1/gonum/stat/distuv"
)

// Run starts
func Run(ps *pubsub.PubSub) {
	// pendingPickups := map[uuid.UUID]time.Timer{}
	shelvedCh := ps.Sub(event.EventTypeShelved)

	p := distuv.Uniform{Min: 2, Max: 10}

	for {
		select {
		case msg := <-shelvedCh:
			// fmt.Print("SHv ")
			newShelvedEvent, ok := msg.(*event.ShelvedEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: Pickup, Shelved, %+v\n", msg)
				continue
			}
			secondsToPickup := p.Rand()
			oneSecond := float64(time.Second)
			pickupEvent := &event.PickupEvent{Dt: newShelvedEvent.Dt, Order: newShelvedEvent.Order}
			timer := time.NewTimer(
				time.Duration(secondsToPickup * oneSecond))
			go pickup(ps, pickupEvent, timer)
		}
	}
}

func pickup(ps *pubsub.PubSub, pickupEvent *event.PickupEvent, timer *time.Timer) {
	<-timer.C
	// fmt.Print("PU^ ")
	ps.Pub(pickupEvent, event.EventTypePickup)

}
