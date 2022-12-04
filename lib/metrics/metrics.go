package metrics

import (
	"github.com/penglongli/gin-metrics/ginmetrics"
)

var monitor *ginmetrics.Monitor

func GetMonitor() *ginmetrics.Monitor {
	monitor := ginmetrics.GetMonitor()

	monitor.SetMetricPath("/__system__/__metrics__")
	monitor.SetSlowTime(10)
	monitor.SetDuration([]float64{0.1, 0.3, 1.2, 5, 10})

	return monitor
}
