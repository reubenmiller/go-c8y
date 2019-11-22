package c8y

import (
	"github.com/reubenmiller/go-c8y/pkg/logger"
)

var Logger *logger.Logger

func init() {
	Logger = logger.NewLogger("c8y")
}

func SilenceLogger() {
	Logger = logger.NewDummyLogger("c8y")
}
