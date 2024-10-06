package testhelpers

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func SetupLogger(t *testing.T) {
	t.Helper()

	// capture the current global logger so it can be restored on test completion.
	globalLogger := log.Logger
	t.Cleanup(func() {
		log.Logger = globalLogger
		zerolog.DefaultContextLogger = nil
	})

	// set up a logger that writes to the test output
	log.Logger = log.
		Output(zerolog.NewTestWriter(t)).
		Level(zerolog.DebugLevel)

	// unless set, the context logger will not log anything
	zerolog.DefaultContextLogger = &log.Logger
}
