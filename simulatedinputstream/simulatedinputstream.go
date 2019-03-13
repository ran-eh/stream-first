package simulatedinputstream

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"stream-first/event"
	"time"

	"github.com/cskr/pubsub"
	"github.com/google/uuid"
	"gonum.org/v1/gonum/stat/distuv"
)

const ordersFile = "data/orders.json"

// Run starts
func Run(ps *pubsub.PubSub) {
	raw, err := ioutil.ReadFile(ordersFile)
	if err != nil {
		log.Fatal(err)
	}
	var data []event.Order
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Fatal(err)
	}

	p := distuv.Exponential{Rate: 3.25}

	for {
		for _, order := range data {
			numSeconds := p.Rand()
			oneSecond := float64(time.Second)
			timer := time.NewTimer(time.Duration(numSeconds * oneSecond))
			now := <-timer.C
			order.ID = uuid.New()
			e := &event.NewOrderEvent{Dt: now, Order: order}
			// fmt.Printf("Publish: %+v\n", e)
			// fmt.Print("NW^ ")
			ps.Pub(e, event.EventTypeNewOrder)
		}
	}
}
