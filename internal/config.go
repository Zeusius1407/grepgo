package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// configName is the file grepgo looks for to supply default flag values.
const configName = ".grepgorc"

// findConfig returns the path of the config file to load, or "" if there is
// none. An explicit path (from --config) is used verbatim; otherwise the
// current directory is preferred over the home directory. A missing file yields
// "" in every case so loadConfig can report it uniformly.
func findConfig(explicit string) string {
	if explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			return explicit
		}
		return ""
	}
	if _, err := os.Stat(configName); err == nil {
		return configName
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, configName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// loadConfig reads "setting = value" lines from the config file and applies
// each to the matching flag, but only when that flag was not set on the command
// line. Precedence is therefore: command line > config file > built-in default.
//
// The format is one setting per line; blank lines and lines starting with '#'
// are ignored, and setting names are the long flag names (e.g. line-number,
// context, color). An explicit --config path that cannot be read is an error;
// the implicit search paths are optional.
func loadConfig(cmd *cobra.Command) error {
	explicit, _ := cmd.Flags().GetString("config")
	path := findConfig(explicit)
	if path == "" {
		if explicit != "" {
			return fmt.Errorf("config file not found: %s", explicit)
		}
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected \"setting = value\"", path, lineNo)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		if key == "config" {
			continue // the config path is not itself configurable
		}
		flag := cmd.Flags().Lookup(key)
		if flag == nil {
			return fmt.Errorf("%s:%d: unknown setting %q", path, lineNo, key)
		}
		if flag.Changed {
			continue // an explicit command-line flag wins over the config
		}
		if err := cmd.Flags().Set(key, val); err != nil {
			return fmt.Errorf("%s:%d: invalid value for %q: %v", path, lineNo, key, err)
		}
	}
	return scanner.Err()
}
