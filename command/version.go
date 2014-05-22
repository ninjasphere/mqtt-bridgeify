package command

import (
	"bytes"
	"fmt"

	"github.com/mitchellh/cli"
)

// VersionCommand is a Command implementation prints the version.
type VersionCommand struct {
	Revision          string
	Version           string
	VersionPrerelease string
	Ui                cli.Ui
}

func (c *VersionCommand) Help() string {
	return ""
}

func (c *VersionCommand) Run(_ []string) int {
	var versionString bytes.Buffer
	fmt.Fprintf(&versionString, "MQTT Proxify v%s", c.Version)

	c.Ui.Output(versionString.String())
	return 0
}

func (c *VersionCommand) Synopsis() string {
	return "Prints the MQTT Proxify version"
}
