package cmd

import (
	"os"
	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/mod/gologger"

	"github.com/projectdiscovery/gologger/levels"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	showVer     bool
	redcProject *redc.RedcProject
)

const BannerArt = `
██████╗  ███████╗ ██████╗   ██████╗ 
 ██╔══██╗ ██╔════╝ ██╔══██╗ ██╔════╝ 
 ██████╔╝ █████╗   ██║  ██║ ██║      
 ██╔══██╗ ██╔══╝   ██║  ██║ ██║      
 ██║  ██║ ███████╗ ██████╔╝ ╚██████╗ 
 ╚═╝  ╚═╝ ╚══════╝ ╚═════╝   ╚═════╝
`

// rootCmd
var rootCmd = &cobra.Command{
	Use:   "redc",
	Short: i18n.T("root_short"),
	Long:  BannerArt + "\n" + i18n.T("root_long"),
	// PersistentPreRun runs before any subcommand
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if showVer {
			return
		}
		// Initialize i18n from environment
		i18n.Init("")
		// Load configuration
		if err := redc.LoadConfig(cfgFile); err != nil {
			gologger.Fatal().Msgf(i18n.Tf("config_load_failed", err.Error()) + "\n")
		}
		if redc.Debug {
			gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
			gologger.Debug().Msgf(i18n.T("debug_mode_enabled"))
		}
		if p, err := redc.ProjectParse(redc.Project, redc.U); err == nil {
			redcProject = p
		} else {
			gologger.Fatal().Msgf(i18n.Tf("project_load_failed", err))
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if showVer {
			gologger.Print().Msgf("%s\nVersion: %s\n", BannerArt, redc.Version)
			return
		}
		// 如果没参数也没flag，打印帮助
		cmd.Help()
	},
}

// Execute 是 main.go 调用的入口
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		gologger.Error().Msgf(err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&showVer, "version", "v", false, i18n.T("flag_version"))
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", i18n.T("flag_config"))
	rootCmd.PersistentFlags().StringVar(&redc.RedcPath, "runpath", "", i18n.T("flag_runpath"))
	rootCmd.PersistentFlags().StringVarP(&redc.U, "user", "u", "system", i18n.T("flag_user"))
	rootCmd.PersistentFlags().StringVar(&redc.Project, "project", "default", i18n.T("flag_project"))
	rootCmd.PersistentFlags().BoolVar(&redc.Debug, "debug", false, i18n.T("flag_debug"))
}
