package ui

import (
	"github.com/cskr/pubsub"
	"stream-first/ui/screen"
	"stream-first/ui/userrequests"
)

func Run(ps *pubsub.PubSub) {
	go userrequests.Run(ps)
	screen.Run(ps)
}
