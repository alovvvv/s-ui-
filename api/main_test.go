package api

import (
	"os"
	"testing"

	"github.com/alireza0/s-ui/logger"

	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
)

func TestMain(m *testing.M) {
	logger.InitLogger(logging.ERROR)
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
