package cli

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/color"
	"github.com/xukonxe/revlay/internal/deployment"
)

// NewStatusCommand creates the `revlay status` command.
func NewStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Displays the current status of the deployment",
		Long:  `Shows the currently active release and the symbolic link structure.`,
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
		return fmt.Errorf("could not get current release: %w", err)
	}

	fmt.Printf("Application: %s\n", color.Cyan(cfg.App.Name))
	fmt.Printf("  - Path: %s\n", cfg.RootPath)
	if currentRelease == "" {
		fmt.Printf("  - Status: %s\n", color.Yellow("No release is currently active"))
	} else {
		fmt.Printf("  - Status: %s\n", color.Green("Active"))
		fmt.Printf("  - Current Release: %s\n", color.Cyan(currentRelease))
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