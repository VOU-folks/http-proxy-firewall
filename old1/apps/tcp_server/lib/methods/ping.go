package methods

import (
	"math/rand"
	"time"
)

func Ping(id string, _ interface{}) interface{} {
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(3000)))
	return "pong"
}
