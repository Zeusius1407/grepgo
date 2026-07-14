# mygrep
A `grep` CLI clone built in Go. The main purpose of building is to familiarise myself with golang and CLI dev concepts.  
## Usage
Clone this repository in your local machine and then run:
```bash
go build -o mygrep
./mygrep [-i] [-v] [-n] [-c] <pattern> <file_path>
```
The tool finds and returns lines in the given file which contain the given pattern.
## Flag description
| Flag | Description |
| --- | --- |
| -h | Help flag. |
| -c | Return the number of times the pattern matches instead of the lines. |
| -i | Ignore case. |
| -n | Give line number of the matching line. |
| -v | Invert matching, return lines which don't contain the pattern. |