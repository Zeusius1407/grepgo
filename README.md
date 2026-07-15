# grepgo

A `grep` clone built in Go. The goal of the project is to get familiar with Go
and CLI development concepts, so the matching engine, terminal detection, and
version stamping are all hand-rolled rather than pulled from libraries.

The tool reads lines from a file (or standard input) and prints those that match
a pattern, supporting a small regex dialect, colored output, recursive search,
context lines, and a config file.

## Installation

Install the latest tagged release directly:

```bash
go install github.com/Zeusius1407/mygrep/cmd/grepgo@latest
```

This produces a `grepgo` binary in `$(go env GOPATH)/bin`.

Or build from a clone:

```bash
git clone https://github.com/Zeusius1407/mygrep.git
cd mygrep
go build -o grepgo ./cmd/grepgo
```

## Usage

```bash
grepgo [flags] <pattern> [file]
```

- If `file` is omitted, `grepgo` reads from **standard input**.
- The pattern is a regular expression (see [Pattern syntax](#pattern-syntax)).

```bash
# match a literal word in a file
grepgo hello notes.txt

# read from stdin
cat notes.txt | grepgo hello

# case-insensitive, with line numbers
grepgo -in hello notes.txt

# recursive search of a directory
grepgo -r hello ./src
```

## Flags

| Short | Long | Argument | Description |
| --- | --- | --- | --- |
| `-i` | `--ignore-case` | | Case-insensitive matching. |
| `-n` | `--line-number` | | Prefix each output line with its 1-based line number. |
| `-v` | `--invert` | | Invert the match: select lines that do **not** match. |
| `-c` | `--count` | | Print only a count of matching lines (suppresses normal and context output). |
| `-r` | `--recursive` | | Recursively search every file under the given directory. Requires a path argument. |
| `-A` | `--after-context` | `NUM` | Print `NUM` lines of trailing context after each match. |
| `-B` | `--before-context` | `NUM` | Print `NUM` lines of leading context before each match. |
| `-C` | `--context` | `NUM` | Print `NUM` lines of context on both sides (shorthand for `-A NUM -B NUM`). |
| | `--color` | `WHEN` | Highlight matches. `WHEN` is `auto` (default), `always`, or `never`. |
| | `--config` | `PATH` | Load defaults from `PATH` instead of the default search locations. |
| `-h` | `--help` | | Show help. |
| | `--version` | | Print the version (`grepgo version …`) and exit. |

## Pattern syntax

Matching is powered by a small recursive-backtracking regex engine
(`internal/regex.go`). Supported syntax:

| Token | Meaning |
| --- | --- |
| `.` | Any single character. |
| `^` | Anchor to the start of the line. |
| `$` | Anchor to the end of the line. |
| `*` | Zero or more of the preceding token. |
| `+` | One or more of the preceding token. |
| `?` | Zero or one of the preceding token. |
| `\x` | Escape: match the literal character `x` (e.g. `\.`, `\*`, `\\`). |

Any other character is a literal. Matching is unanchored (the pattern may match
any substring) unless anchored with `^` / `$`.

```bash
grepgo 'he.*o'  notes.txt   # he, then anything, then o
grepgo 'hel+o'  notes.txt   # one or more 'l'
grepgo '^From:' mail.txt    # lines starting with "From:"
grepgo 'end\.'  notes.txt   # a literal "end."
```

> Note: this engine is for learning and does not provide the linear-time
> guarantees of Go's standard `regexp` package; pathological patterns can
> backtrack heavily.

## Colored output

Matches are highlighted in bold red when `--color` is enabled.

- `auto` (default) — colorize only when standard output is a terminal, detected
  via an `isatty`-style check (`internal/terminal.go`). Output is left plain
  when piped or redirected to a file.
- `always` — always emit color escapes.
- `never` — never colorize.

Context lines and inverted (`-v`) matches are never highlighted.

## Context lines

`-A`, `-B`, and `-C` mirror `grep`'s context behavior:

- Matching lines are separated from their fields by `:`; context lines use `-`
  (e.g. `notes.txt:12:match` vs `notes.txt-11-context`).
- Non-adjacent groups of output are separated by a `--` line.
- Overlapping or adjacent context windows are merged without duplicating lines.
- `-C NUM` fills in for `-A`/`-B` only when those were not given explicitly.
- `-c` (count) ignores context.

```bash
grepgo -C2 -n ERROR app.log
```

## Configuration file (`.grepgorc`)

`grepgo` reads default flag values from a `.grepgorc` file. The first file that
exists is used, in order:

1. the path given by `--config`
2. `./.grepgorc` (current directory)
3. `$HOME/.grepgorc` (home directory)

Precedence is **command line > config file > built-in defaults**: a flag set on
the command line always wins over the same setting in the file.

**Format** — one `setting = value` per line. Blank lines and lines beginning
with `#` are ignored. Setting names are the **long flag names**, values are
`true`/`false` for booleans, numbers for counts, and strings otherwise.

```ini
# ~/.grepgorc
# always number lines and show two lines of context
line-number = true
context     = 2
color       = always
```

Errors are reported with the file and line number and exit with status `2`:
an unknown setting, an invalid value (e.g. a non-numeric `context`), or a
missing `--config` path.

## Versioning

The version is derived from the Git repository at build time via
`runtime/debug.ReadBuildInfo()` — no `-ldflags` required (`internal/version.go`):

- Installed from a tag (e.g. `go install …/grepgo@v1.3.0`) → reports that tag.
- A local build → reports the short commit hash, with `-dirty` appended when the
  working tree has uncommitted changes.

```bash
grepgo --version
# grepgo version v1.3.0
```

## Exit codes

Following `grep`'s convention:

| Code | Meaning |
| --- | --- |
| `0` | At least one line matched. |
| `1` | No lines matched. |
| `2` | An error occurred (e.g. file not readable, bad config, missing `-r` path). |

## Project layout

```
cmd/grepgo/main.go    # entry point; imports internal/ (package cmd) and calls cmd.Execute()
internal/root.go      # cobra command, flag wiring, search/output logic
internal/regex.go     # recursive-backtracking regex engine
internal/terminal.go  # isatty-style terminal detection
internal/version.go   # version string from build metadata
internal/config.go    # .grepgorc loading
```
