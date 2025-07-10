package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/xukonxe/revlay/internal/deployment"
	"github.com/xukonxe/revlay/internal/i18n"
	"github.com/xukonxe/revlay/internal/ssh"
)

var releasesCmd = &cobra.Command{
	Use:   "releases",
	Short: "",
	Long:  ``,
	RunE: runReleases,
}

func init() {
	// Update command descriptions when config is initialized
	cobra.OnInitialize(func() {
		t := i18n.T()
		releasesCmd.Short = t.ReleasesShortDesc
		releasesCmd.Long = t.ReleasesLongDesc
	})
}

func runReleases(cmd *cobra.Command, args []string) error {
	t := i18n.T()
	
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Create SSH client
	sshConfig := &ssh.Config{
		Host:     cfg.Server.Host,
		User:     cfg.Server.User,
		Port:     cfg.Server.Port,
		Password: cfg.Server.Password,
		KeyFile:  cfg.Server.KeyFile,
	}

	client, err := ssh.NewClient(sshConfig)
	if err != nil {
		return fmt.Errorf(t.ErrorSSHConnect, err)
	}
	defer client.Close()

	// Create deployer
	deployer := deployment.NewDeployer(cfg, client)

	// List releases
	releases, err := deployer.ListReleases()
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	if len(releases) == 0 {
		fmt.Println(t.ReleasesNoReleases)
		return nil
	}

	// Display releases in a table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RELEASE\tTIMESTAMP\tCURRENT\tPATH")
	fmt.Fprintln(w, "-------\t---------\t-------\t----")

	for _, release := range releases {
		current := ""
		if release.Current {
			current = "✓"
		}
		
		timestamp := ""
		if !release.Timestamp.IsZero() {
			timestamp = release.Timestamp.Format("2006-01-02 15:04:05")
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", release.Name, timestamp, current, release.Path)
	}

	w.Flush()

	// Show summary
	fmt.Printf("\nTotal releases: %d\n", len(releases))
	fmt.Printf("Keep releases: %d\n", cfg.App.KeepReleases)
	
	if len(releases) > cfg.App.KeepReleases {
		fmt.Printf("⚠️  %d old releases will be cleaned up on next deployment\n", len(releases)-cfg.App.KeepReleases)
	}

	return nil
}