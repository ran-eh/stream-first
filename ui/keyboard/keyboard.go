package keyboard

import (
	"bufio"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

// Run runs
func Run(ch chan rune) {
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
		ch <- rune
		// fmt.Printf("Rune: %v\n", rune)
		if rune == 3 {
			break
		}
	}
}
