package main

import (
	"stream-first/pickup"
	"stream-first/shelf"
	"stream-first/shelflife"
	input "stream-first/simneworders"
	display "stream-first/ui"

	"github.com/cskr/pubsub"
)

func main() {
	ps := pubsub.New(1000)

	//go keyboard.Run(kbdCh)
	// go displayRun(ps)
	go display.Run(ps)
	go input.Run(ps)
	go shelf.Run(ps)
	go shelflife.Run(ps)
	go pickup.Run(ps)

	// Block forever
	select {}
}
