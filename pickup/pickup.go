package pickup

import (
	"fmt"
	"stream-first/common"
	"time"

	"github.com/cskr/pubsub"
	"github.com/orcaman/concurrent-map"
	"gonum.org/v1/gonum/stat/distuv"
)

// Run starts
func Run(ps *pubsub.PubSub) {
	pendingPickups := cmap.New()
	shelvedCh := ps.Sub(common.EventTypeShelved)
	expiredCh := ps.Sub(common.EventTypeExpired)

	// Pickup interval is random.
	p := distuv.Uniform{Min: 2, Max: 10}

	for {
		select {
		case msg := <-shelvedCh:
			newShelvedEvent, ok := msg.(*common.ShelvedEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: Pickup, Shelved, %+v\n", msg)
				continue
			}
			secondsToPickup := p.Rand()
			oneSecond := float64(time.Second)
			pickupEvent := &common.PickupEvent{Dt: newShelvedEvent.Dt, Order: newShelvedEvent.Order}
			timer := time.NewTimer(
				time.Duration(secondsToPickup * oneSecond))
			pendingPickups.Set(newShelvedEvent.Order.ID.String(), timer)
			go pickup(ps, pickupEvent, timer)
		case msg := <-expiredCh:
			expiredEvent, ok := msg.(*common.ShelvedEvent)
			if !ok {
				// Error
				fmt.Printf("could not coerce to event: Pickup, Expired, %+v\n", msg)
				continue
			}
			orderIDStr := expiredEvent.Order.ID.String()
			if timerInterface, ok := pendingPickups.Get(orderIDStr); ok {
				timer := timerInterface.(time.Timer)
				timer.Stop()
				pendingPickups.Remove(orderIDStr)
			}
		}
	}
}

func pickup(ps *pubsub.PubSub, pickupEvent *common.PickupEvent, timer *time.Timer) {
	<-timer.C
	ps.Pub(pickupEvent, common.EventTypePickup)
}
