package agent

import "github.com/wolfeidau/usage"

const JSON_RPC_VERSION = "2.0"

type metricsEvent struct {
	Params         []interface{} `json:"params"`
	Time           int64         `json:"time"`
	JsonRpcVersion string        `json:"jsonrpc"`
}

type metricUsage struct {
	Memory uint64  `json:"memory"`
	Cpu    float64 `json:"cpu"`
}

type MetricService struct {
	processMonitor *usage.ProcessMonitor
}

func CreateMetricService() *MetricService {
	return &MetricService{processMonitor: usage.CreateProcessMonitor()}
}

func (rs *MetricService) buildMetricsRequest() *metricsEvent {

	var args []interface{} = make([]interface{}, 2)

	memUsage := rs.processMonitor.GetMemoryUsage()
	cpuUsage := rs.processMonitor.GetCpuUsage()

	args[0] = "mqtt-bridgeify"
	args[1] = &metricUsage{Memory: memUsage.Resident, Cpu: cpuUsage.Total}

	return &metricsEvent{
		Params:         args,
		Time:           int64(usage.UnixTimeMs()), // unix time in milliseconds
		JsonRpcVersion: JSON_RPC_VERSION,
	}

}
