package common

import (
	"fmt"
	"time"
)

// Definitions for posting diagnostic messages.

// Severity constants
var (
	Error   = "ERROR"
	Info    = "INFO"
	Warning = "WARN"
)

// Diag posts a diagnostic message
func Diag(ps PubsubInterface, serviceName string, severity string, message string, error error) {
	ps.Pub(&DiagEvent{
		Dt:          time.Now(),
		ServiceName: serviceName,
		Severity:    severity,
		Message:     message,
		Error:       error,
	}, DiagTopic)

}

// String converts a diagnostic message into a format appropriate for log presentation
func (e DiagEvent) String() string {
	var message string
	if e.Error != nil {
		message = fmt.Sprint(e.Error)
	} else {
		message = e.Message
	}
	return fmt.Sprintf("%v: %v: %v: %v",
		e.Dt.Format("01-02 15:04:05"), e.Severity, e.ServiceName, message)
}

// Creates a diagnostic message for type coercion failure.  Used when subscribers process
// events and the event is not of the expected type.
func CoerceErrorMessage(source interface{}, target interface{}) string {
	return fmt.Sprintf("Unexpected message type: %T received, %T expected", source, target)
}
