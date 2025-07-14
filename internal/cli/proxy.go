package cli

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/proxy"
)

// NewProxyCommand creates the `revlay proxy` command.
func NewProxyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Runs the built-in TCP proxy for zero-downtime deployments",
		Long: `Runs the built-in TCP proxy.
This command should be run as a persistent service (e.g., using systemd).
It listens on the 'proxy_port' and forwards traffic to the active application port.
It watches a state file for changes to perform seamless traffic switching.`,
		RunE: runProxy,
		Args: cobra.NoArgs,
	}
	return cmd
}

func runProxy(cmd *cobra.Command, args []string) error {
	cfgFile, _ := cmd.Flags().GetString("config")
	cfg, err := loadConfig(cfgFile)
	if err != nil {
		return err
	}

	if cfg.Deploy.Mode != "zero_downtime" || cfg.Service.ProxyPort == 0 {
		return fmt.Errorf("proxy command is only available for 'zero_downtime' mode with 'proxy_port' configured")
	}

	log.Println(color.Cyan("Starting Revlay proxy..."))

	stateFile := cfg.GetActivePortPath()
	initialPort := cfg.Service.Port // Default to main port on first run

	manager := proxy.NewManager(cfg.Service.ProxyPort, initialPort, stateFile)

	// This is a blocking call that runs the proxy server indefinitely.
	if err := manager.Start(); err != nil {
		return fmt.Errorf("failed to start proxy manager: %w", err)
	}

	return nil
}
