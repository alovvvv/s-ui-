package sub

import (
	"os"
	"testing"

	"github.com/alireza0/s-ui/logger"

	"github.com/op/go-logging"
)

// The subscription code logs on every malformed-input path it now tolerates,
// and an uninitialised s-ui logger is a nil pointer. The real binary calls this
// at startup; the test binary has to do it too.
func TestMain(m *testing.M) {
	logger.InitLogger(logging.ERROR)
	os.Exit(m.Run())
}
