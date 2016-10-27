package cmd

import (
	"fmt"
	"os"

	"strings"
	"time"

	"github.com/getcarina/carina/client"
	"github.com/getcarina/carina/common"
	"github.com/getcarina/carina/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cxt *context

func newCarinaCommand() *cobra.Command {
	// Global application context
	cxt = &context{}

	// Local command context
	var opts struct {
		Version bool
	}

	var cmd = &cobra.Command{
		Use:   "carina",
		Short: "Create and interact with clusters on both Rackspace Public and Private Clouds",
		Long:  "Create and interact with clusters on both Rackspace Public and Private Clouds",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Version {
				writeVersion()
				return nil
			}
			fmt.Print(cmd.UsageString())
			return nil
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cxt.initialize()
			if err != nil {
				return err
			}

			return checkIsLatest()
		},
	}

	// Local command flags
	cmd.Flags().BoolVarP(&opts.Version, "version", "v", false, "Show the application version")
	cmd.Flags().MarkHidden("version")

	//
	// Global application flags
	//
	cmd.PersistentFlags().StringVar(&cxt.ConfigFile, "config", "", "config file (default is CARINA_HOME/config.toml)")
	cmd.PersistentFlags().BoolVar(&cxt.CacheEnabled, "cache", true, "Cache API tokens and update times")
	cmd.PersistentFlags().BoolVar(&cxt.Debug, "debug", false, "Print additional debug messages to stdout")
	cmd.PersistentFlags().BoolVar(&cxt.Silent, "silent", false, "Do not print to stdout")

	// Account flags
	cmd.PersistentFlags().StringVar(&cxt.Profile, "profile", "", "Use saved credentials from a profile")
	cmd.PersistentFlags().BoolVar(&cxt.ProfileDisabled, "no-profile", false, "Ignore profiles and use flags and/or environment variables only")
	cmd.PersistentFlags().StringVar(&cxt.Username, "username", "", "Username [CARINA_USERNAME/RS_USERNAME/OS_USERNAME]")
	cmd.PersistentFlags().StringVar(&cxt.APIKey, "api-key", "", "Public Cloud API Key [CARINA_APIKEY/RS_API_KEY]")
	cmd.PersistentFlags().StringVar(&cxt.Password, "password", "", "Private Cloud Password [OS_PASSWORD]")
	cmd.PersistentFlags().StringVar(&cxt.Project, "project", "", "Private Cloud Project Name [OS_PROJECT_NAME]")
	cmd.PersistentFlags().StringVar(&cxt.Domain, "domain", "", "Private Cloud Domain Name [OS_DOMAIN_NAME]")
	cmd.PersistentFlags().StringVar(&cxt.Region, "region", "", "Private Cloud Region Name [OS_REGION_NAME]")
	cmd.PersistentFlags().StringVar(&cxt.AuthEndpoint, "auth-endpoint", "", "Private Cloud Authentication endpoint [OS_AUTH_URL]")
	cmd.PersistentFlags().StringVar(&cxt.Endpoint, "endpoint", "", "Custom API endpoint [CARINA_ENDPOINT/OS_ENDPOINT]")
	cmd.PersistentFlags().StringVar(&cxt.CloudType, "cloud", "", "The cloud type: public or private")

	// Hide local development flags
	cmd.PersistentFlags().MarkHidden("config")
	cmd.PersistentFlags().MarkHidden("cache")
	cmd.PersistentFlags().MarkHidden("endpoint")

	// Don't show usage on errors
	cmd.SilenceUsage = true

	authHelp := `Authentication:
The user credentials are used to automatically detect the cloud with which the cli should communicate. First, it looks for the Rackspace Public Cloud environment variables, such as CARINA_USERNAME/CARINA_APIKEY or RS_USERNAME/RS_API_KEY. Then it looks for Rackspace Private Cloud environment variables, such as OS_USERNAME/OS_PASSWORD. Use --cloud flag to explicitly select a cloud.

In the following example, the detected cloud is 'private' because --password is specified:
    carina --username bob --password ilovepuppies --project admin --auth-endpoint http://example.com/auth/v3 ls

In the following example, the detected cloud is 'public' because --apikey is specified:
    carina --username bob --apikey abc123 ls

In the following example, 'private' will be used, even when the Rackspace Public Cloud environment variables are present, because --cloud is specified:
    carina --cloud private ls

Profiles:
Credentials can be saved under a profile name in CARINA_HOME/config.toml, and then used with the --profile flag. If --profile is not specified, and the config file contains a profile named 'default', it will be used when no credential flags are provided.

Below is a sample config file:

    [default]
    cloud="public"
    username="alicia"
    apikey="abc123"

    [dev]
    cloud="private"
    username-var="OS_USERNAME"
    password-var="OS_PASSWORD"
    auth-endpoint-var="OS_AUTH_URL"
    project-var="OS_PROJECT_NAME"
    domain-var="OS_PROJECT_DOMAIN_NAME"
    region-var="OS_REGION_NAME"

In the following example, the default profile is used because no other credentials were explicitly provided:
    carina ls

In the following example, the dev profile is used:
    carina --profile dev ls

See https://getcarina.com/docs/tutorials/carina-cli for additional documentation, FAQ and examples.
`

	baseDir, err := client.GetCredentialsDir()
	if err != nil {
		panic(err)
	}
	envHelp := fmt.Sprintf(`Environment Variables:
  CARINA_HOME
    directory that stores your cluster tokens and credentials
    current setting: %s
`, baseDir)
	cmd.SetUsageTemplate(fmt.Sprintf("%s\n%s\n\n%s", cmd.UsageTemplate(), envHelp, authHelp))

	cobra.OnInitialize(initConfig)

	cmd.AddCommand(
		newAutoScaleCommand(),
		newBashCompletionCmd(),
		newCreateCommand(),
		newCredentialsCommand(),
		newDeleteCommand(),
		newEnvCommand(),
		newGetCommand(),
		newGrowCommand(),
		newClustersCommand(),
		newTemplatesCommand(),
		newQuotasCommand(),
		newRebuildCommand(),
		newVersionCommand(),
	)
	return cmd
}

// Execute the root carina command
func Execute() {
	rootCmd := newCarinaCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cxt.ConfigFile != "" {
		common.Log.WriteDebug("CONFIG: --config %s", cxt.ConfigFile)
		viper.SetConfigFile(cxt.ConfigFile)

		err := viper.ReadInConfig()
		if err != nil {
			common.Log.WriteError("Unable to read configuration file: %s", err, cxt.ConfigFile)
		}
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("$HOME/.carina")

		err := viper.ReadInConfig()
		if err != nil {
			common.Log.WriteDebug("CONFIG: %s", cxt.ConfigFile)
		}
	}
}

func checkIsLatest() error {
	if !cxt.CacheEnabled {
		return nil
	}

	ok, err := shouldCheckForUpdate()
	if !ok {
		return err
	}
	common.Log.WriteDebug("Checking for newer releases of the carina cli...")

	rel, err := version.LatestRelease()
	if err != nil {
		common.Log.WriteWarning("# Unable to fetch information about the latest release of %s. %s\n.", os.Args[0], err)
		return nil
	}
	common.Log.WriteDebug("Latest: %s", rel.TagName)

	latest, err := version.ExtractSemver(rel.TagName)
	if err != nil {
		common.Log.WriteWarning("# Trouble parsing latest tag (%v): %s", rel.TagName, err)
		return nil
	}

	current, err := version.ExtractSemver(version.Version)
	if err != nil {
		common.Log.WriteWarning("# Trouble parsing current tag (%v): %s", version.Version, err)
		return nil
	}
	common.Log.WriteDebug("Installed: %s", version.Version)

	if latest.Greater(current) {
		common.Log.WriteWarning("# A new version of the Carina client is out, go get it!")
		common.Log.WriteWarning("# You're on %v and the latest is %v", current, latest)
		common.Log.WriteWarning("# https://github.com/getcarina/carina/releases")
	}

	return nil
}

func shouldCheckForUpdate() (bool, error) {
	lastCheck := cxt.Client.Cache.LastUpdateCheck

	// If we last checked recently, don't check again
	if lastCheck.Add(12 * time.Hour).After(time.Now()) {
		common.Log.Debug("Skipping check for a new release as we have already checked recently")
		return false, nil
	}

	err := cxt.Client.Cache.SaveLastUpdateCheck(time.Now())
	if err != nil {
		return false, err
	}

	if strings.Contains(version.Version, "-dev") || version.Version == "" {
		common.Log.Debug("Skipping check for new release because this is a developer build")
		return false, nil
	}

	return true, nil
}