package agent

type Agent struct {
	conf *Config
}

func createAgent(conf *Config) *Agent {
	return &Agent{conf: conf}
}

func (a *Agent) start() error {
	return nil
}

func (a *Agent) stop() error {
	return nil
}
