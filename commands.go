package main

import (
	"os"
	"os/signal"

	"github.com/mitchellh/cli"
	"github.com/ninjablocks/mqtt-bridgeify/command"
	"github.com/ninjablocks/mqtt-bridgeify/command/agent"
)

var Commands map[string]cli.CommandFactory

func init() {
	ui := &cli.BasicUi{Writer: os.Stdout}

	Commands = map[string]cli.CommandFactory{

		"agent": func() (cli.Command, error) {
			return &agent.Command{
				Ui:         ui,
				ShutdownCh: make(chan struct{}),
			}, nil
		},

		"version": func() (cli.Command, error) {
			return &command.VersionCommand{
				Version: Version,
				Ui:      ui,
			}, nil
		},
	}
}

func makeShutdownCh() <-chan struct{} {
	resultCh := make(chan struct{})

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		for {
			<-signalCh
			resultCh <- struct{}{}
		}
	}()

	return resultCh
}
