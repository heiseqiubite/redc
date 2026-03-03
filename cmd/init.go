package cmd

import (
	"os"
	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/mod/gologger"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: i18n.T("init_short"),
	Run: func(cmd *cobra.Command, args []string) {
		redc.RedcLog("执行初始化")
		gologger.Info().Msg(i18n.T("init_running"))

		dirs, err := redc.ScanTemplateDirs(redc.TemplateDir, redc.MaxTfDepth)
		if err != nil {
			gologger.Error().Msgf(i18n.Tf("init_scan_failed", err))
		}
		for _, v := range dirs {
			if err := redc.TfInit(v); err != nil {
				gologger.Error().Msgf(i18n.Tf("init_scene_failed", v, err))
			} else {
				gologger.Info().Msgf(i18n.Tf("init_scene_done", v))
			}
		}
	},
}

// completionCmd 生成命令补全脚本
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: i18n.T("completion_short"),
	Long:  i18n.T("completion_long"),
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(initCmd)
}
