package ordersender

// The Order Sender component simulates the a source of incoming orders.  It reads orders
// from the sample file, and publishes them at random intervals.

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"stream-first/common"
	"stream-first/ui/userrequests"
	"time"

	"github.com/cskr/pubsub"
	"github.com/google/uuid"
	"gonum.org/v1/gonum/stat/distuv"
)

//noinspection NonAsciiCharacters
const (
	serviceName = "OrderSender"
	ordersFile  = "data/orders.json"
	// Yes, Go does support non ascii identifiers :)
	λ = 3.25
)

var paused bool

// Run simulates a new order source.  It reads orders from a data file, and publishes them in random intervals.
func Run(ps *pubsub.PubSub) {
	userRequestCh := ps.Sub(common.UserRequestTopic)
	// Allow time for other components to subscribe before starting to publish.
	time.Sleep(common.Seconds(common.SchedulerDelay))

	common.Diag(ps, serviceName, common.Info, "Service started.", nil)
	go pubOrders(ps)
	for {
		msg := <-userRequestCh
		userRequest, ok := msg.(string)
		if !ok {
			panic("could not coerce")
		}
		switch userRequest {
		case userrequests.PauseIncomingOrders:
			paused = true
		case userrequests.ResumeIncomingOrders:
			paused = false
		}
	}
}

func pubOrders(ps *pubsub.PubSub) {
	raw, err := ioutil.ReadFile(ordersFile)
	if err != nil {
		log.Fatal(err)
	}
	var data []common.Order
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Fatal(err)
	}

	p := distuv.Exponential{Rate: λ}

	for {
		for _, order := range data {
			numSeconds := p.Rand()
			timer := time.NewTimer(common.Seconds(numSeconds))
			now := <-timer.C
			order.ID = uuid.New()
			e := &common.NewOrderEvent{Dt: now, Order: order}
			if !paused {
				ps.Pub(e, common.NewOrderTopic)
			}
		}
	}
}
