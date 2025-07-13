package cli

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/i18n"
)

// NewStatusCommand creates the `revlay status` command.
func NewStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: i18n.T().StatusShortDesc,
		Long:  i18n.T().StatusLongDesc,
		RunE:  runStatus,
	}
	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfgFile, _ := cmd.Flags().GetString("config")
	cfg, err := loadConfig(cfgFile)
	if err != nil {
		return err
	}

	deployer := deployment.NewLocalDeployer(cfg)
	currentRelease, err := deployer.GetCurrentRelease()
	if err != nil {
		return fmt.Errorf("could not get current release: %v", err)
	}

	fmt.Printf(i18n.T().StatusAppName+"\n", color.Cyan(cfg.App.Name))
	fmt.Printf(i18n.T().StatusDeployPath+"\n", cfg.RootPath)
	if currentRelease == "" {
		fmt.Printf("  - Status: %s\n", color.Yellow(i18n.T().StatusNoRelease))
	} else {
		fmt.Printf("  - Status: %s\n", color.Green("Active"))
		fmt.Printf(i18n.T().StatusCurrentRelease+"\n", color.Cyan(currentRelease))
	}

	fmt.Println("\nDirectory Details:")
	lsCmd := exec.Command("ls", "-l", cfg.RootPath)
	output, err := lsCmd.Output()
	if err != nil {
		fmt.Printf("  - Could not get directory details: %v\n", err)
	} else {
		fmt.Printf("\n%s\n", string(output))
	}

	return nil
}
