package userrequests

import (
	"bufio"
	"fmt"
	"github.com/cskr/pubsub"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"stream-first/common"
	"time"
)

// This package captures keyboard events, and generates corresponding user request events.
const (
	serviceName = "UI/UserRequests"
)

// User request events
const (
	QuitRequest          = "quit"
	PausePickup          = "pausePickup"
	ResumePickup         = "resumePickup"
	PauseIncomingOrders  = "pauseIncomingOrders"
	ResumeIncomingOrders = "resumeIncomingOrders"
)

// Handled key presses
var (
	quitRunes           = map[rune]bool{'q': true, 'Q': true, 3: true}
	togglePickupRunes   = map[rune]bool{'p': true, 'P': true}
	toggleIncomingRunes = map[rune]bool{'i': true, 'I': true}
)

var RequestedState = struct {
	PickUpPaused         bool
	IncomingOrdersPaused bool
}{}

func Run(ps *pubsub.PubSub) {
	// Allow time for other components to subscribe before starting to publish.
	time.Sleep(common.Seconds(common.SchedulerDelay))
	common.Diag(ps, serviceName, common.Info, "Service started.", nil)

	// Set terminal to raw mode
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("error: %+v\n", err)
	}

	// restore appears not to work on linux, `stty echo cooked` is required after exiting the program.
	defer func() { _ = terminal.Restore(0, oldState) }()

	reader := bufio.NewReader(os.Stdin)

	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			panic(err)
		}
		var userRequest string
		if quitRunes[r] {
			userRequest = QuitRequest
		}
		if togglePickupRunes[r] {
			if RequestedState.PickUpPaused {
				userRequest = ResumePickup
			} else {
				userRequest = PausePickup
			}
			RequestedState.PickUpPaused = !RequestedState.PickUpPaused
		}
		if toggleIncomingRunes[r] {
			if RequestedState.IncomingOrdersPaused {
				userRequest = ResumeIncomingOrders
			} else {
				userRequest = PauseIncomingOrders
			}
			RequestedState.IncomingOrdersPaused = !RequestedState.IncomingOrdersPaused
		}
		if userRequest != "" {
			ps.Pub(userRequest, common.UserRequestTopic)
		}
	}
}
