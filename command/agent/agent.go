package agent

import "time"

//
// Pulls together the bridge, a cached state configuration and the bus.
//
type Agent struct {
	conf    *Config
	bridge  *Bridge
	eventCh chan statusEvent
}

func createAgent(conf *Config) *Agent {
	return &Agent{conf: conf, bridge: createBridge(conf)}
}

// TODO load the existing configuration on startup and start the bridge if needed
func (a *Agent) start() error {

	return nil
}

// stop all the things.
func (a *Agent) stop() error {

	return nil
}

func (a *Agent) startBridge(connect *connectRequest) {
	a.bridge.start(connect.Url, connect.Token)
}

// save the state of the bridge then disconnect it
func (a *Agent) stopBridge(disconnect *disconnectRequest) {
	a.bridge.stop()
}

func (a *Agent) getStatus() statsEvent {
	return statsEvent{
		Connected:  a.bridge.Connected,
		Configured: a.bridge.Configured,
		Count:      a.bridge.Counter,
		Time:       time.Now().Unix(),
	}
}
