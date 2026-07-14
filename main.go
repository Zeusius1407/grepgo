/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Define flags manually
	ignoreCase := flag.Bool("i", false, "case insensitive matching")
	lineNumbers := flag.Bool("n", false, "show line numbers")
	invert := flag.Bool("v", false, "invert match")
	count := flag.Bool("c", false, "count matches")

	flag.Parse() // parses os.Args, strips out the flags

	// After flags are parsed, remaining positional args are here
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: mygrep [-i] [-n] [-v] [-c] <pattern> [file]")
		os.Exit(2)
	}

	pattern := args[0]
	if *ignoreCase {
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
	counter := 0;
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		check := line
		if *ignoreCase {
			check = strings.ToLower(check)
		}

		matched := strings.Contains(check, pattern)
		if *invert {
			matched = !matched
		}

		if matched {
			found = true
			counter += 1
			if !*count {
				if *lineNumbers {
					fmt.Printf("%d: %s\n", lineNum, line)
				} else {
					fmt.Println(line)
				}
			}
		}
	}
	if *count {
		fmt.Printf("%d\n", counter)
	}
	if found {
		os.Exit(0)
	}
	os.Exit(1)
}