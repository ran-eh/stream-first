package pickup_test

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"stream-first/common"
	"stream-first/mocks"
	"stream-first/pickup"
	"stream-first/ui/userrequests"
	"testing"
	"time"
)

var (
	testOrder       = common.Order{Name: "something yummy", Temp: "some temp", ID: uuid.New(), DecayRate: 10, ShelfLife: 30}
	secondsToPickup = 0.2231
	secondsToExpire = 0.1231
)

func TestRun0(t *testing.T) {
	rand := &mocks.MockRand{MockResult: secondsToPickup}
	t.Run("A pickup event is fired after the required duration", func(t *testing.T) {
		ps, shelvedCh, expiredCh, userRequestCh, stopCh := initParams()
		go pickup.Run0(rand, ps, shelvedCh, expiredCh, userRequestCh, stopCh)

		ps.On("Pub", mock.Anything, mock.Anything)
		pubShelved(shelvedCh)
		// Wait a bit longer than pickup time to allow for gorouting scheduling time
		time.Sleep(common.Seconds(secondsToPickup + common.SchedulerDelay))
		pickupAt := time.Now()
		ps.AssertNumberOfCalls(t, "Pub", 1)
		ps.AssertCalled(t, "Pub", mock.MatchedBy(func(msg interface{}) bool {
			pickupEvent := msg.(*common.PickupEvent)
			return pickupEvent.Order == testOrder && pickupAt.After(pickupEvent.Dt)
		}), []string{common.PickupTopic})
		stopCh <- true
	})
	t.Run("An expire event circumvents the pickup event", func(t *testing.T) {
		ps, shelvedCh, expiredCh, userRequestCh, stopCh := initParams()
		go pickup.Run0(rand, ps, shelvedCh, expiredCh, userRequestCh, stopCh)

		ps.On("Pub", mock.Anything, mock.Anything)
		pubShelved(shelvedCh)
		time.Sleep(common.Seconds(common.SchedulerDelay))
		pubExpired(expiredCh)
		time.Sleep(common.Seconds(common.SchedulerDelay))
		ps.AssertNotCalled(t, "Pub", mock.Anything, mock.Anything)
		stopCh <- true
	})
	t.Run("An pickup event is not generated when the service is paused", func(t *testing.T) {
		ps, shelvedCh, expiredCh, userRequestCh, stopCh := initParams()
		go pickup.Run0(rand, ps, shelvedCh, expiredCh, userRequestCh, stopCh)

		ps.On("Pub", mock.Anything, mock.Anything)

		userRequestCh <- userrequests.PausePickup
		pubShelved(shelvedCh)
		time.Sleep(common.Seconds(secondsToExpire + common.SchedulerDelay))
		pubExpired(expiredCh)
		time.Sleep(common.Seconds(secondsToPickup + common.SchedulerDelay))
		ps.AssertNotCalled(t, "Pub", mock.Anything, mock.Anything)
		stopCh <- true
	})
	t.Run("When a shelved event arrives when service is paused, a pickup event fires after service resumed", func(t *testing.T) {
		ps, shelvedCh, expiredCh, userRequestCh, stopCh := initParams()
		go pickup.Run0(rand, ps, shelvedCh, expiredCh, userRequestCh, stopCh)

		ps.On("Pub", mock.Anything, mock.Anything)

		userRequestCh <- userrequests.PausePickup
		pubShelved(shelvedCh)
		userRequestCh <- userrequests.ResumePickup
		// Wait a bit longer than pickup time to allow for gorouting scheduling time
		time.Sleep(common.Seconds(secondsToPickup + common.SchedulerDelay))
		pickupAt := time.Now()
		ps.AssertNumberOfCalls(t, "Pub", 1)
		ps.AssertCalled(t, "Pub", mock.MatchedBy(func(msg interface{}) bool {
			pickupEvent := msg.(*common.PickupEvent)
			return pickupEvent.Order == testOrder && pickupAt.After(pickupEvent.Dt)
		}), []string{common.PickupTopic})
		stopCh <- true
	})
}

func pubShelved(shelvedCh chan interface{}) {
	shelvedAt := time.Now()
	shelvedEvent := &common.ShelvedEvent{Order: testOrder, Shelf: "a shelf", Dt: shelvedAt}
	shelvedCh <- shelvedEvent
}

func pubExpired(expiredCh chan interface{}) {
	expiredAt := time.Now()
	expiredEvent := &common.ExpiredEvent{Order: testOrder, Dt: expiredAt}
	expiredCh <- expiredEvent
}

func initParams() (ps *mocks.MockPubsub,
	shelvedCh chan interface{}, expiredCh chan interface{}, userRequestCh chan interface{}, stopCh chan bool) {
	ps = &mocks.MockPubsub{}
	shelvedCh = make(chan interface{})
	expiredCh = make(chan interface{})
	userRequestCh = make(chan interface{})
	stopCh = make(chan bool)
	return
}
