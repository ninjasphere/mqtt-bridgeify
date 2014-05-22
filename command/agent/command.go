package agent

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/logutils"
	"github.com/mitchellh/cli"
)

/*
 Holds all the context for a running agent
*/
type Command struct {
	Ui         cli.Ui
	ShutdownCh <-chan struct{}
	args       []string
	logFilter  *logutils.LevelFilter
	logger     *log.Logger
	agent      *Agent
}

type Config struct {
	Token    string
	CloudUrl string
	LocalUrl string
	Debug bool
}

func (c *Config) IsDebug() bool{
	return c.Debug
}

func (c *Command) readConfig() *Config {
	var cmdConfig Config
	cmdFlags := flag.NewFlagSet("agent", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }
	cmdFlags.StringVar(&cmdConfig.Token, "token", "", "ninja token")
	cmdFlags.StringVar(&cmdConfig.LocalUrl, "localurl", "tcp://localhost:1883", "cloud url to connect to")
	cmdFlags.StringVar(&cmdConfig.CloudUrl, "cloudurl", "ssl://dev.ninjasphere.co:8883", "cloud url to connect to")
	cmdFlags.BoolVar(&cmdConfig.Debug, "debug", false, "enable debug")

	if err := cmdFlags.Parse(c.args); err != nil {
		return nil
	}

	// Ensure we have a token
	if cmdConfig.Token == "" {
		c.Ui.Error("Must specify token using -token")
		return nil
	}

	return &cmdConfig
}

func (c *Command) handleSignals(config *Config) int {
	var sig os.Signal
	select {
	case <-c.ShutdownCh:
		sig = os.Interrupt
	}
	c.Ui.Output(fmt.Sprintf("Caught signal: %v", sig))

	return 0
}

func (c *Command) Run(args []string) int {
	c.Ui = &cli.PrefixedUi{
		OutputPrefix: "==> ",
		InfoPrefix:   "    ",
		ErrorPrefix:  "==> ",
		Ui:           c.Ui,
	}

	c.args = args
	config := c.readConfig()
	if config == nil {
		return 1
	}
	c.args = args

	c.Ui.Output("MQTT bridgeify agent running!")
	c.Ui.Info("Token loaded: " + config.Token)
	c.Ui.Info("Local url: " + config.LocalUrl)
	c.Ui.Info("Cloud url: " + config.CloudUrl)

	c.agent = createAgent(config)

	if err := c.agent.start(); err != nil {
		c.Ui.Error(fmt.Sprintf("error starting agent %s", err))
	}

	return c.handleSignals(config)
}

func (c *Command) Synopsis() string {
	return "Runs a MQTT bridgeify agent"
}

func (c *Command) Help() string {
	helpText := `
Usage: mqtt-bridgeify agent [options]

  Starts the MQTT bridgeify agent and runs until an interrupt is received.

Options:

  -localurl=tcp://localhost:1883           URL for the local broker.
  -localurl=ssl://dev.ninjasphere.co:8883  URL for the remote broker.
  -debug                                   Enables debug output.
  -token=                                  The ninja sphere token.
`
	return helpText
}
