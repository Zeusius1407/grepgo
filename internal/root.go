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
	"strconv"
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
	before      int  // lines of context to print before a match (-B)
	after       int  // lines of context to print after a match (-A)
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

// bufLine holds a line and its 1-based number so before-context can be printed
// once a later line matches.
type bufLine struct {
	num  int
	text string
}

// searchReader scans one reader for matching lines and prints them, including
// any requested before/after context lines. It returns whether at least one
// line matched. When opts.count is set it prints only the match count (prefixed
// with the file name when opts.withName is set) and ignores context.
func searchReader(reader *os.File, name string, opts searchOpts) bool {
	scanner := bufio.NewScanner(reader)
	lineNum := 0
	found := false
	counter := 0

	hasContext := opts.before > 0 || opts.after > 0
	lastPrinted := 0   // number of the most recently printed line
	hasPrinted := false // whether anything has been printed yet (for "--")
	pending := 0        // remaining after-context lines still to print
	var before []bufLine

	// printLine emits one output line. isMatch selects grep's separators: ":"
	// for matching lines, "-" for context lines (used between the file name,
	// the line number, and the text). A "--" divider is written between
	// non-adjacent groups when context is in effect, mirroring grep.
	printLine := func(num int, text string, isMatch bool) {
		if hasContext && hasPrinted && num > lastPrinted+1 {
			fmt.Println("--")
		}
		sep := "-"
		if isMatch {
			sep = ":"
		}
		var b strings.Builder
		if opts.withName {
			b.WriteString(name)
			b.WriteString(sep)
		}
		if opts.lineNumbers {
			b.WriteString(strconv.Itoa(num))
			b.WriteString(sep)
		}
		b.WriteString(text)
		fmt.Println(b.String())
		lastPrinted = num
		hasPrinted = true
	}

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
				// Emit buffered before-context that hasn't been printed yet.
				for _, bl := range before {
					if bl.num > lastPrinted {
						printLine(bl.num, bl.text, false)
					}
				}
				out := line
				// Coloring highlights the matched text, so it only makes sense
				// for lines selected *because* they matched (not with -v).
				if opts.color && !opts.invert {
					out = highlight(line, check, opts)
				}
				printLine(lineNum, out, true)
				pending = opts.after
			}
		} else if pending > 0 && !opts.count {
			// After-context following the most recent match.
			printLine(lineNum, line, false)
			pending--
		}

		// Remember this line as potential before-context for a later match.
		if opts.before > 0 {
			before = append(before, bufLine{lineNum, line})
			if len(before) > opts.before {
				before = before[len(before)-opts.before:]
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
	Use:   "grepgo [flags] <pattern> [file]",
	Short: "A grep clone written in Go",
	Long: `grepgo is a CLI tool that searches files for lines that match the given pattern.

Configuration file:
  grepgo reads default flag values from a .grepgorc file, using the first that
  exists (in order):
    1. the path given by --config
    2. ./.grepgorc      (current directory)
    3. $HOME/.grepgorc  (home directory)

  Command-line flags override the config file, which overrides the built-in
  defaults. The file holds one "setting = value" per line; blank lines and
  lines beginning with '#' are ignored. Setting names are the long flag names.

  Example .grepgorc:
    # always number lines and show two lines of context
    line-number = true
    context     = 2
    color       = always`,
	Args: cobra.RangeArgs(1, 2),
	// Load .grepgorc defaults before Run reads the flag values.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return loadConfig(cmd)
	},
	// Matching lines are printed by Run itself; the exit code is set here so
	// that "no match" reports 1 the way grep does.
	Run: func(cmd *cobra.Command, args []string) {
		ignoreCase, _ := cmd.Flags().GetBool("ignore-case")
		lineNumbers, _ := cmd.Flags().GetBool("line-number")
		invert, _ := cmd.Flags().GetBool("invert")
		count, _ := cmd.Flags().GetBool("count")
		recursive, _ := cmd.Flags().GetBool("recursive")
		colorWhen, _ := cmd.Flags().GetString("color")

		// Context flags: -A/-B set after/before independently; -C sets both
		// unless the more specific flag was given explicitly.
		after, _ := cmd.Flags().GetInt("after-context")
		before, _ := cmd.Flags().GetInt("before-context")
		context, _ := cmd.Flags().GetInt("context")
		if context > 0 {
			if !cmd.Flags().Changed("after-context") {
				after = context
			}
			if !cmd.Flags().Changed("before-context") {
				before = context
			}
		}
		if after < 0 {
			after = 0
		}
		if before < 0 {
			before = 0
		}

		opts := searchOpts{
			pattern:     args[0],
			ignoreCase:  ignoreCase,
			lineNumbers: lineNumbers,
			invert:      invert,
			count:       count,
			before:      before,
			after:       after,
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
	// Version is derived from the Git repository at build time (see version.go).
	rootCmd.Version = buildVersion()

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
	rootCmd.Flags().IntP("after-context", "A", 0, "print NUM lines of trailing context after matches")
	rootCmd.Flags().IntP("before-context", "B", 0, "print NUM lines of leading context before matches")
	rootCmd.Flags().IntP("context", "C", 0, "print NUM lines of output context (before and after)")
	rootCmd.Flags().String("config", "", "path to config file (default: ./.grepgorc or $HOME/.grepgorc)")
}
