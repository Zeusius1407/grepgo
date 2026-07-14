/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mygrep [flags] <pattern> [file]",
	Short: "A grep clone written in Go",
	Long: `mygrep is a CLI tool that searches files for lines that match to the given pattern`,
	Args: cobra.RangeArgs(1, 2),
	// Matching lines are printed by Run itself; the exit code is set here so
	// that "no match" reports 1 the way grep does.
	Run: func(cmd *cobra.Command, args []string) {
		ignoreCase, _ := cmd.Flags().GetBool("ignore-case")
		lineNumbers, _ := cmd.Flags().GetBool("line-number")
		invert, _ := cmd.Flags().GetBool("invert")
		count, _ := cmd.Flags().GetBool("count")

		pattern := args[0]
		if ignoreCase {
			pattern = strings.ToLower(pattern)
		}

		// Decide input source: file if given, else stdin
		var reader *os.File
		if len(args) >= 2 {
			f, err := os.Open(args[1])
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(2)
			}
			defer f.Close()
			reader = f
		} else {
			reader = os.Stdin
		}

		scanner := bufio.NewScanner(reader)
		lineNum := 0
		found := false
		counter := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			check := line
			if ignoreCase {
				check = strings.ToLower(check)
			}

			matched := strings.Contains(check, pattern)
			if invert {
				matched = !matched
			}

			if matched {
				found = true
				counter++
				if !count {
					if lineNumbers {
						fmt.Printf("%d: %s\n", lineNum, line)
					} else {
						fmt.Println(line)
					}
				}
			}
		}
		if count {
			fmt.Printf("%d\n", counter)
		}
		if found {
			os.Exit(0)
		}
		os.Exit(1)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(2)
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
