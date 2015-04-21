package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
)

// StatusCommand is a Command that outputs the status of whether
// Vault is sealed or not as well as HA information.
type StatusCommand struct {
	Meta
}

func (c *StatusCommand) Run(args []string) int {
	flags := c.Meta.FlagSet("status", FlagSetDefault)
	flags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := flags.Parse(args); err != nil {
		return 1
	}

	client, err := c.Client()
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error initializing client: %s", err))
		return 2
	}

	sealStatus, err := client.Sys().SealStatus()
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error checking seal status: %s", err))
		return 2
	}
	c.Ui.Output(fmt.Sprintf(
		"Sealed: %v\n"+
			"Key Shares: %d\n"+
			"Key Threshold: %d\n"+
			"Unseal Progress: %d",
		sealStatus.Sealed,
		sealStatus.N,
		sealStatus.T,
		sealStatus.Progress))

	// Mask the 'Vault is sealed' error, since this means HA is enabled,
	// but that we cannot query for the leader since we are sealed.
	leaderStatus, err := client.Sys().Leader()
	if err != nil && strings.Contains(err.Error(), "Vault is sealed") {
		leaderStatus = &api.LeaderResponse{HAEnabled: true}
		err = nil
	}
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error checking leader status: %s", err))
		return 2
	}

	// Output if HA is enabled
	c.Ui.Output("")
	c.Ui.Output(fmt.Sprintf("High-Availability Enabled: %v", leaderStatus.HAEnabled))
	if leaderStatus.HAEnabled {
		if sealStatus.Sealed {
			c.Ui.Output("\tMode: sealed")
		} else {
			mode := "standby"
			if leaderStatus.IsSelf {
				mode = "active"
			}
			c.Ui.Output(fmt.Sprintf("\tMode: %s", mode))

			if leaderStatus.LeaderAddress == "" {
				leaderStatus.LeaderAddress = "<none>"
			}
			c.Ui.Output(fmt.Sprintf("\tLeader: %s", leaderStatus.LeaderAddress))
		}
	}

	if sealStatus.Sealed {
		return 1
	} else {
		return 0
	}
}

func (c *StatusCommand) Synopsis() string {
	return "Outputs status of whether Vault is sealed and if HA mode is enabled"
}

func (c *StatusCommand) Help() string {
	helpText := `
Usage: vault status [options]

  Outputs the state of the Vault, sealed or unsealed and if HA is enabled.

  This command outputs whether or not the Vault is sealed. The exit
  code also reflects the seal status (0 unsealed, 1 sealed, 2+ error).

General Options:

  -address=TODO           The address of the Vault server.

  -ca-cert=path           Path to a PEM encoded CA cert file to use to
                          verify the Vault server SSL certificate.

  -ca-path=path           Path to a directory of PEM encoded CA cert files
                          to verify the Vault server SSL certificate. If both
                          -ca-cert and -ca-path are specified, -ca-path is used.

  -insecure               Do not verify TLS certificate. This is highly
                          not recommended. This is especially not recommended
                          for unsealing a vault.

`
	return strings.TrimSpace(helpText)
}
