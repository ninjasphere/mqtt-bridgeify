package agent

import "runtime"

//
// Pulls together the bridge, a cached state configuration and the bus.
//
type Agent struct {
	conf     *Config
	bridge   *Bridge
	memstats *runtime.MemStats
	eventCh  chan statusEvent
}

func createAgent(conf *Config) *Agent {
	return &Agent{conf: conf, bridge: createBridge(conf), memstats: &runtime.MemStats{}}
}

// TODO load the existing configuration on startup and start the bridge if needed
func (a *Agent) start() error {

	return nil
}

// stop all the things.
func (a *Agent) stop() error {

	return nil
}

func (a *Agent) startBridge(connect *connectRequest) error {
	return a.bridge.start(connect.Url, connect.Token)
}

// save the state of the bridge then disconnect it
func (a *Agent) stopBridge(disconnect *disconnectRequest) error {
	return a.bridge.stop()
}

func (a *Agent) getStatus() statsEvent {

	var lastError string

	if a.bridge.LastError != nil {
		lastError = a.bridge.LastError.Error()
	}

	runtime.ReadMemStats(a.memstats)

	return statsEvent{
		LastError:  lastError,
		Alloc:      a.memstats.Alloc,
		HeapAlloc:  a.memstats.HeapAlloc,
		TotalAlloc: a.memstats.TotalAlloc,
		Connected:  a.bridge.Connected,
		Configured: a.bridge.Configured,
		Count:      a.bridge.Counter,
	}
}
