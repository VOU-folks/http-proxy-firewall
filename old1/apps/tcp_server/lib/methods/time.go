package methods

import (
	"time"
)

func Time(id string, _ interface{}) interface{} {
	return time.Now().UTC().String()
}
