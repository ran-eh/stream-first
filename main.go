package main

import (
	"fmt"
	"github.com/cskr/pubsub"
	"os"
	"stream-first/common"
	input "stream-first/ordersender"
	"stream-first/pickup"
	"stream-first/shelf"
	"stream-first/shelflife"
	"stream-first/ui"
	"stream-first/ui/userrequests"
)

const serviceName = "Main"

// Launch all services and wait for the quit user request.
func main() {
	ps := pubsub.New(1000)

	userCh := ps.Sub(common.UserRequestTopic)
	go ui.Run(ps)
	go input.Run(ps)
	go shelf.Run(ps)
	go shelflife.Run(ps)
	go pickup.Run(ps)

	for {
		msg := <-userCh
		userRequest, ok := msg.(string)
		if !ok {
			common.Diag(ps, serviceName, common.Error, common.CoerceErrorMessage(msg, userRequest), nil)
		}

		switch userRequest {
		case userrequests.QuitRequest:
			fmt.Printf("\r\n")
			os.Exit(0)
		}
	}
}
