package keyboard

import (
	"bufio"
	"fmt"
	"github.com/cskr/pubsub"
	"os"
	"stream-first/common"

	"golang.org/x/crypto/ssh/terminal"
)

// Run runs
func Run(ps pubsub.PubSub) {
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("error: %+v\n", err)
	}
	fmt.Printf("made it: %+v", oldState)

	defer terminal.Restore(0, oldState)

	reader := bufio.NewReader(os.Stdin)

	for {
		rune, _, err := reader.ReadRune()
		if err != nil {
			panic(err)
		}
		ps.Pub(rune, common.EventTypeKeyboard)
	}
}
