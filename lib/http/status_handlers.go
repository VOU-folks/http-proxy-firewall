package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"http-proxy-firewall/lib/db/cookie"
)

var startedAt time.Time

func init() {
	startedAt = time.Now()
}

func timeDiff(a time.Time, b time.Time) string {
	return ""
	/*
		diff := b.Sub(a).Seconds()
		hours := int(diff / 3600)
		minutes := diff - hours*3600

		minutes := (diff - seconds) % 60
		hours := (diff - 60*minutes - seconds)
	*/
}

func Status(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"started": startedAt.Format(time.RFC3339),
		// "uptime":     timeDiff(startedAt, time.Now()),
		"cookieSize": cookie.Size(),
		"cookieCap":  cookie.Cap(),
	})
}
