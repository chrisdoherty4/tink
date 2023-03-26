package failure_test

import (
	"errors"
	"testing"

	"github.com/tinkerbell/tink/internal/agent/failure"
)

func TestReason(t *testing.T) {
	for _, tc := range []struct {
		Name          string
		Error         error
		ExpectReason  string
		ExpectMessage string
	}{
		{
			Name:          "NewReason",
			Error:         failure.NewReason("new reason", "NewReason"),
			ExpectReason:  "NewReason",
			ExpectMessage: "new reason",
		},
		{
			Name:          "WithReason",
			Error:         failure.WithReason(errors.New("with reason"), "WithReason"),
			ExpectReason:  "WithReason",
			ExpectMessage: "with reason",
		},
		{
			Name:          "CustomError",
			Error:         customError{Reason: "CustomReason", Message: "custom reason"},
			ExpectReason:  "CustomReason",
			ExpectMessage: "custom reason",
		},
		{
			Name:          "NoReason",
			Error:         errors.New("no reason"),
			ExpectMessage: "no reason",
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			reason, ok := failure.Reason(tc.Error)

			if tc.ExpectReason != "" {
				if !ok {
					t.Fatal("Expected a reason but didn't find one")
				}

				if reason != tc.ExpectReason {
					t.Fatalf("[Reason] Expected: %v; Received: %v", tc.ExpectReason, reason)
				}
			} else {
				if ok {
					t.Fatal("Reason unexpectedly returned true for second parameter")
				}

				if reason != "" {
					t.Fatalf("Reason unexpectedly returned a reason: %v", reason)
				}
			}

			// Ensure the error message is the same as the expected message. This is primarily
			// for cases where we construct something with a reason and need to check the message
			// comes out as expected.
			if tc.Error.Error() != tc.ExpectMessage {
				t.Fatalf("[Message] Expected: %v; Received: %v", tc.ExpectMessage, tc.Error.Error())
			}
		})
	}
}

type customError struct {
	Reason, Message string
}

func (c customError) Error() string {
	return c.Message
}

func (c customError) FailureReason() string {
	return c.Reason
}
