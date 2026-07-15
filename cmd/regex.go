package cmd

// A tiny recursive-backtracking regex engine. Supported syntax:
//
//	.        any single character
//	^        anchor to start of text
//	$        anchor to end of text
//	*        zero or more of the preceding token
//	+        one or more of the preceding token
//	?        zero or one of the preceding token
//	\x       escape: match the literal character x (e.g. \. \* \\)
//
// Everything else is a literal character. Matching is "unanchored" like grep:
// the pattern may match any substring of the text unless anchored with ^/$.
//
// The core matcher returns the number of bytes consumed (or -1 for no match)
// so that callers can recover the matched span, not just a yes/no answer.

// regexMatch reports whether pattern matches anywhere in text.
func regexMatch(pattern, text string) bool {
	_, _, ok := regexFind(pattern, text)
	return ok
}

// regexFind returns the byte range [start,end) of the leftmost match of pattern
// in text, and whether a match was found.
func regexFind(pattern, text string) (start, end int, ok bool) {
	anchoredStart := false
	if len(pattern) > 0 && pattern[0] == '^' {
		anchoredStart = true
		pattern = pattern[1:]
	}
	// Try each start position, including the empty tail so "$" and "" can match
	// at the end of the text.
	for i := 0; i <= len(text); i++ {
		if n := matchHere(pattern, text[i:]); n >= 0 {
			return i, i + n, true
		}
		if anchoredStart {
			break
		}
	}
	return 0, 0, false
}

// matchHere returns the number of bytes of text consumed by matching pattern at
// the start of text, or -1 if pattern does not match here.
func matchHere(pattern, text string) int {
	if pattern == "" {
		return 0
	}
	if pattern == "$" {
		if text == "" {
			return 0
		}
		return -1
	}

	// Read one token off the front of the pattern; an escape is two bytes.
	tokenLen := 1
	if pattern[0] == '\\' && len(pattern) >= 2 {
		tokenLen = 2
	}
	token := pattern[:tokenLen]
	rest := pattern[tokenLen:]

	// A quantifier, if present, applies to the token just read.
	if len(rest) > 0 {
		switch rest[0] {
		case '*':
			return matchStar(token, rest[1:], text)
		case '+':
			return matchPlus(token, rest[1:], text)
		case '?':
			return matchQuestion(token, rest[1:], text)
		}
	}

	if text != "" && matchToken(token, text[0]) {
		if n := matchHere(rest, text[1:]); n >= 0 {
			return n + 1
		}
	}
	return -1
}

// matchToken reports whether the single token matches the byte c.
func matchToken(token string, c byte) bool {
	if len(token) == 2 && token[0] == '\\' {
		return token[1] == c
	}
	return token[0] == '.' || token[0] == c
}

// matchStar matches zero or more of token, then rest. It tries the fewest
// repetitions first and backtracks by consuming more, so rest still gets a
// chance to match after each expansion.
func matchStar(token, rest, text string) int {
	for i := 0; ; i++ {
		if n := matchHere(rest, text[i:]); n >= 0 {
			return i + n
		}
		if i >= len(text) || !matchToken(token, text[i]) {
			return -1
		}
	}
}

// matchPlus matches one or more of token, then rest.
func matchPlus(token, rest, text string) int {
	if text == "" || !matchToken(token, text[0]) {
		return -1
	}
	if n := matchStar(token, rest, text[1:]); n >= 0 {
		return n + 1
	}
	return -1
}

// matchQuestion matches zero or one of token, then rest.
func matchQuestion(token, rest, text string) int {
	if text != "" && matchToken(token, text[0]) {
		if n := matchHere(rest, text[1:]); n >= 0 {
			return n + 1
		}
	}
	return matchHere(rest, text) // zero occurrences
}
