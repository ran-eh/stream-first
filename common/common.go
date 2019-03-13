package common

// This package contains logic used by multiple components
import (
	"time"
)

// The time in seconds to wait for the scheduler to visit other services
var SchedulerDelay = 0.001

// Used to make pubsub operation testable via mocks
type PubsubInterface interface {
	Pub(interface{}, ...string)
	Sub(...string) chan interface{}
}

// Allows mocking random calls.
type RandInterface interface {
	Rand() float64
}

func Seconds(seconds float64) time.Duration {
	OneSecond := float64(time.Second)
	return time.Duration(seconds * OneSecond)

}
