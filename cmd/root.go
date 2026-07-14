/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)



// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mygrep",
	Short: "A grep clone written in Go",
	Long: `mygrep is a CLI tool that searches files for lines that match to the given pattern`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) { 
		ignoreCase, _ := cmd.Flags().GetBool("ignore-case")
		lineNumbers, _ := cmd.Flags().GetBool("line-number")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.mygrep.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("ignore-case", "i", false, "case insensitive matching")
	rootCmd.Flags().BoolP("line-number", "n", false, "show line numbers")
	rootCmd.Flags().BoolP("invert", "v", false, "invert match")
	rootCmd.Flags().BoolP("count", "c", false, "only show count of matches")
	rootCmd.Flags().BoolP("recursive", "r", false, "search directories recursively")
}


