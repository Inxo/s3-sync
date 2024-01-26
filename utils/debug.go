package utils

import (
	"os"
	"strconv"
)

func IsDebug() bool {
	debug, err := strconv.ParseBool(os.Getenv("APP_DEBUG"))
	if err != nil {
		return false
	}
	return debug
}
