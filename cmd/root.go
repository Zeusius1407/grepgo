/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"github.com/spf13/cobra"
)

// searchOpts bundles the flag-derived options passed down to searchReader.
type searchOpts struct {
	pattern     string
	ignoreCase  bool
	lineNumbers bool
	invert      bool
	count       bool
	withName    bool // prefix output with the file name (recursive / multi-file)
	color       bool // highlight matched text with ANSI escapes
}

// ANSI escape sequences used to highlight matches, mirroring grep's default.
const (
	colorMatch = "\x1b[1;31m" // bold red
	colorReset = "\x1b[0m"
)

// highlight wraps every non-overlapping match of the pattern in ANSI color
// codes. It operates on positions found in check (which may be lower-cased for
// case-insensitive matching) but emits bytes from the original line, so it only
// applies when the two align byte-for-byte; otherwise line is returned as-is.
func highlight(line, check string, opts searchOpts) string {
	if len(check) != len(line) {
		return line
	}
	var b strings.Builder
	i := 0
	for i < len(check) {
		s, e, ok := regexFind(opts.pattern, check[i:])
		if !ok {
			break
		}
		s += i
		e += i
		if s == e { // zero-width match: nothing to color, avoid looping forever
			break
		}
		b.WriteString(line[i:s])
		b.WriteString(colorMatch)
		b.WriteString(line[s:e])
		b.WriteString(colorReset)
		i = e
	}
	b.WriteString(line[i:])
	return b.String()
}

// searchReader scans one reader for matching lines and prints them. It returns
// whether at least one line matched. When opts.count is set it prints the count
// (prefixed with the file name when opts.withName is set) instead of the lines.
func searchReader(reader *os.File, name string, opts searchOpts) bool {
	scanner := bufio.NewScanner(reader)
	lineNum := 0
	found := false
	counter := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		check := line
		if opts.ignoreCase {
			check = strings.ToLower(check)
		}

		matched := regexMatch(opts.pattern, check)
		if opts.invert {
			matched = !matched
		}

		if matched {
			found = true
			counter++
			if !opts.count {
				prefix := ""
				if opts.withName {
					prefix = name + ":"
				}
				out := line
				// Coloring highlights the matched text, so it only makes sense
				// for lines selected *because* they matched (not with -v).
				if opts.color && !opts.invert {
					out = highlight(line, check, opts)
				}
				if opts.lineNumbers {
					fmt.Printf("%s%d:%s\n", prefix, lineNum, out)
				} else {
					fmt.Printf("%s%s\n", prefix, out)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	if opts.count {
		if opts.withName {
			fmt.Printf("%s:%d\n", name, counter)
		} else {
			fmt.Printf("%d\n", counter)
		}
	}
	return found
}

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
		recursive, _ := cmd.Flags().GetBool("recursive")
		colorWhen, _ := cmd.Flags().GetString("color")

		opts := searchOpts{
			pattern:     args[0],
			ignoreCase:  ignoreCase,
			lineNumbers: lineNumbers,
			invert:      invert,
			count:       count,
		}
		if ignoreCase {
			opts.pattern = strings.ToLower(opts.pattern)
		}

		// Decide whether to colorize. "auto" (the default) uses our isTerminal
		// check so colors only appear on a real terminal, not when piped or
		// redirected to a file.
		switch colorWhen {
		case "always":
			opts.color = true
		case "never":
			opts.color = false
		default: // "auto"
			opts.color = isTerminal(os.Stdout)
		}

		if recursive {
			// A path is required to know what to walk; there's no
			// recursive search of stdin.
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "error: -r requires a path argument")
				os.Exit(2)
			}
			opts.withName = true
			found := false
			hadError := false
			err := filepath.WalkDir(args[1], func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					fmt.Fprintln(os.Stderr, "error:", err)
					hadError = true
					return nil
				}
				if d.IsDir() {
					return nil
				}
				f, err := os.Open(path)
				if err != nil {
					fmt.Fprintln(os.Stderr, "error:", err)
					hadError = true
					return nil
				}
				defer f.Close()
				if searchReader(f, path, opts) {
					found = true
				}
				return nil
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(2)
			}
			if hadError {
				os.Exit(2)
			}
			if found {
				os.Exit(0)
			}
			os.Exit(1)
		}

		// Decide input source: file if given, else stdin
		var reader *os.File
		name := "(standard input)"
		if len(args) >= 2 {
			f, err := os.Open(args[1])
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(2)
			}
			defer f.Close()
			reader = f
			name = args[1]
		} else {
			reader = os.Stdin
		}

		found := searchReader(reader, name, opts)
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
	rootCmd.Flags().BoolP("recursive", "r", false, "recursively search directories")
	rootCmd.Flags().String("color", "auto", "highlight matches: auto|always|never")
}
