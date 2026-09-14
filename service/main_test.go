package service

import (
	"os"
	"testing"

	"github.com/alireza0/s-ui/logger"

	"github.com/op/go-logging"
)

// TestMain initialises the s-ui logger before any test runs. Constructing a
// core.Core builds the protocol registries, and those log through it (the naive
// stub reports that naive is absent); an uninitialised logger is a nil pointer
// and panics. core/main_test.go does the same for the same reason.
func TestMain(m *testing.M) {
	logger.InitLogger(logging.ERROR)
	os.Exit(m.Run())
}
