package test

import (
	"log"
)

func BuildLogger() *log.Logger {
	return log.New(nil, "", 0)
}
