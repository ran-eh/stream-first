package pickup

// The pickup service generates pickups for newly shelved orders, and cancels outstanding pickups for
// expire orders.

import (
	"stream-first/common"
	"stream-first/ui/userrequests"
	"time"

	"github.com/cskr/pubsub"
	"github.com/orcaman/concurrent-map"
	"gonum.org/v1/gonum/stat/distuv"
)

const (
	serviceName = "Pickup"
)

// When set, pickups are paused.
var paused bool

func Run(ps *pubsub.PubSub) {

	shelvedCh := ps.Sub(common.ShelvedTopic)
	expiredCh := ps.Sub(common.ExpiredTopic)
	userRequestCh := ps.Sub(common.UserRequestTopic)

	// Allow time for other components to subscribe before starting to publish.
	time.Sleep(common.Seconds(common.SchedulerDelay))
	common.Diag(ps, serviceName, common.Info, "Service started.", nil)

	p := distuv.Uniform{Min: 2, Max: 10}

	Run0(p, ps, shelvedCh, expiredCh, userRequestCh, nil)
}

// Run0 is a testable version of the service.  It allows injecting mocks for pub/sub and Rand calls.
func Run0(p common.RandInterface, ps common.PubsubInterface,
	shelvedCh chan interface{}, expiredCh chan interface{}, userRequestCh chan interface{}, stopCh chan bool) {

	// A thread-safe map storing timers for pending order pickups.
	pendingPickups := cmap.New()

	for {
		select {
		case msg := <-shelvedCh:
			e, ok := msg.(*common.ShelvedEvent)
			if !ok {
				common.Diag(ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			secondsToPickup := p.Rand()
			timer := time.NewTimer(common.Seconds(secondsToPickup))
			pickupEvent := &common.PickupEvent{Dt: e.Dt, Order: e.Order}
			pendingPickups.Set(e.Order.ID.String(), timer)
			go pickup(ps, pickupEvent, timer, p)
		case msg := <-expiredCh:
			e, ok := msg.(*common.ExpiredEvent)
			if !ok {
				common.Diag(ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			orderIDStr := e.Order.ID.String()
			if timerInterface, ok := pendingPickups.Get(orderIDStr); ok {
				timer := timerInterface.(*time.Timer)
				timer.Stop()
				pendingPickups.Remove(orderIDStr)
			}
		case msg := <-userRequestCh:
			e, ok := msg.(string)
			if !ok {
				common.Diag(ps, serviceName, common.Error, common.CoerceErrorMessage(msg, e), nil)
				continue
			}
			switch e {
			case userrequests.PausePickup:
				paused = true
			case userrequests.ResumePickup:
				paused = false
			}
		case <-stopCh:
			break
		}
	}
}

func pickup(ps common.PubsubInterface, pickupEvent *common.PickupEvent, timer *time.Timer, r common.RandInterface) {
	<-timer.C
	for paused {
		secondsToPickup := r.Rand()
		time.Sleep(common.Seconds(secondsToPickup))
	}
	ps.Pub(pickupEvent, common.PickupTopic)
}
