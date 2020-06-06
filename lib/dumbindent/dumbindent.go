// Copyright 2020 The Wuffs Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// ----------------

// Package dumbindent formats C (and C-like) programs.
//
// It is similar in concept to pretty-printers like `indent` or `clang-format`.
// It is much dumber (it will not add line breaks or otherwise re-flow lines of
// code, not to fit within an 80 character limit nor for any other reason) but
// it can therefore be much, much faster at the basic task of automatically
// indenting nested blocks. The output isn't 'perfect', but it's usually
// sufficiently readable if the input already has sensible line breaks.
//
// See `cmd/dumbindent/main.go` in this repository for an example where
// `dumbindent` was 80 times faster than `clang-format`.
//
// There are no configuration options (e.g. tabs versus spaces).
package dumbindent

import (
	"bytes"
)

// 'Constants', but their type is []byte, not string.
var (
	backTick  = []byte("`")
	externC   = []byte("extern \"C\"")
	spaces    = []byte("                                ")
	starSlash = []byte("*/")
)

// hangingBytes is a look-up table for updating the hanging variable.
var hangingBytes = [256]bool{
	'=':  true,
	'\\': true,
}

// FormatBytes formats the C (or C-like) program in src, appending the result
// to dst, and returns that longer slice.
//
// It is valid to pass a dst slice (such as nil) whose spare capacity (not
// including its existing elements) is too short to hold the formatted program.
// In this case, a new slice will be allocated and returned.
func FormatBytes(dst []byte, src []byte) []byte {
	if len(src) == 0 {
		return dst
	} else if len(dst) == 0 {
		dst = make([]byte, 0, len(src)+(len(src)/2))
	}

	nBraces := 0       // The number of unbalanced '{'s.
	nParens := 0       // The number of unbalanced '('s.
	openBrace := false // Whether the previous non-blank line ends with '{'.
	hanging := false   // Whether the previous non-blank line ends with '=' or '\\'.
	blankLine := false // Whether the previous line was blank.

	for line, remaining := src, []byte(nil); len(src) > 0; src = remaining {
		src = trimLeadingWhiteSpace(src)
		line, remaining = src, nil
		if i := bytes.IndexByte(line, '\n'); i >= 0 {
			line, remaining = line[:i], line[i+1:]
		}
		lineLength := len(line)

		// Collapse 2 or more consecutive blank lines into 1. Also strip any
		// blank lines:
		//  - immediately after a '{',
		//  - immediately before a '}',
		//  - at the end of file.
		if len(line) == 0 {
			blankLine = true
			continue
		}
		if blankLine {
			blankLine = false
			if !openBrace && (line[0] != '}') {
				dst = append(dst, '\n')
			}
		}

		// Preprocessor lines (#ifdef, #pragma, etc) are never indented.
		//
		// The '{' and '}' for an `extern "C"` are also special cased not to
		// change indentation inside the block. This assumes that the closing
		// brace is followed by a `// extern "C"` comment.
		if (line[0] == '#') ||
			((line[0] == 'e') && bytes.HasPrefix(line, externC)) ||
			((line[0] == '}') && bytes.HasSuffix(line, externC)) {
			line = trimTrailingWhiteSpace(line)
			dst = append(dst, line...)
			dst = append(dst, '\n')
			openBrace = false
			hanging = lastNonWhiteSpace(line) == '\\'
			continue
		}

		// Account for leading '}'s before we print the line's indentation.
		closeBraces := 0
		for ; (closeBraces < len(line)) && line[closeBraces] == '}'; closeBraces++ {
		}
		nBraces -= closeBraces

		// When debugging, uncomment these (and import "strconv") to prefix
		// every non-blank line with the state.
		//
		// dst = append(dst, strconv.Itoa(nBraces)...)
		// dst = append(dst, ',')
		// dst = append(dst, strconv.Itoa(nParens)...)
		// if openBrace {
		//   dst = append(dst, '{')
		// } else {
		//   dst = append(dst, ':')
		// }
		// if hanging {
		//   dst = append(dst, '=')
		// } else {
		//   dst = append(dst, ':')
		// }

		// Output a certain number of spaces to rougly approximate the
		// "clang-format -style=Chromium" indentation style.
		indent := 0
		if nBraces > 0 {
			indent += 2 * nBraces
		}
		if (nParens > 0) || hanging {
			indent += 4
		}
		if (indent >= 2) && isLabel(line) {
			indent -= 2
		}
		for indent > 0 {
			n := indent
			if n > len(spaces) {
				n = len(spaces)
			}
			dst = append(dst, spaces[:n]...)
			indent -= n
		}

		// Output the leading '}'s.
		dst = append(dst, line[:closeBraces]...)
		line = line[closeBraces:]

		// Adjust the state according to the braces and parentheses within the
		// line (except for those in comments and strings).
		last := lastNonWhiteSpace(line)
	loop:
		for {
			for i, c := range line {
				switch c {
				case '{':
					nBraces++
				case '}':
					nBraces--
				case '(':
					nParens++
				case ')':
					nParens--

				case '/':
					if (i + 1) >= len(line) {
						break
					}
					if line[i+1] == '/' {
						// A slash-slash comment. Skip the rest of the line.
						last = lastNonWhiteSpace(line[:i])
						break loop
					} else if line[i+1] == '*' {
						// A slash-star comment.
						dst = append(dst, line[:i+2]...)
						restOfLine := line[i+2:]
						restOfSrc := src[lineLength-len(restOfLine):]
						dst, line, remaining = handleRaw(dst, restOfSrc, starSlash)
						last = lastNonWhiteSpace(line)
						continue loop
					}

				case '"', '\'':
					// A cooked string, whose contents are backslash-escaped.
					suffix := skipCooked(line[i+1:], c)
					dst = append(dst, line[:len(line)-len(suffix)]...)
					line = suffix
					continue loop

				case '`':
					// A raw string.
					dst = append(dst, line[:i+1]...)
					restOfLine := line[i+1:]
					restOfSrc := src[lineLength-len(restOfLine):]
					dst, line, remaining = handleRaw(dst, restOfSrc, backTick)
					last = lastNonWhiteSpace(line)
					continue loop
				}
			}
			break loop
		}
		openBrace = last == '{'
		hanging = hangingBytes[last]

		// Output the line (minus any trailing space).
		line = trimTrailingWhiteSpace(line)
		dst = append(dst, line...)
		dst = append(dst, "\n"...)
	}
	return dst
}

// trimLeadingWhiteSpace converts "\t  foo bar " to "foo bar ".
func trimLeadingWhiteSpace(s []byte) []byte {
	for (len(s) > 0) && ((s[0] == ' ') || (s[0] == '\t')) {
		s = s[1:]
	}
	return s
}

// trimTrailingWhiteSpace converts "\t  foo bar " to "\t  foo bar".
func trimTrailingWhiteSpace(s []byte) []byte {
	for (len(s) > 0) && ((s[len(s)-1] == ' ') || (s[len(s)-1] == '\t')) {
		s = s[:len(s)-1]
	}
	return s
}

// isLabel returns whether s looks like "foo:" or "bar_baz:;".
func isLabel(s []byte) bool {
	for (len(s) > 0) && (s[len(s)-1] == ';') {
		s = s[:len(s)-1]
	}
	if (len(s) < 2) || (s[len(s)-1] != ':') {
		return false
	}
	s = s[:len(s)-1]
	for _, c := range s {
		switch {
		case ('0' <= c) && (c <= '9'):
		case ('A' <= c) && (c <= 'Z'):
		case ('a' <= c) && (c <= 'z'):
		case c == '_':
		default:
			return false
		}
	}
	return true
}

// lastNonWhiteSpace returns the 'z' in "abc xyz  ". It returns '\x00' if s
// consists entirely of spaces or tabs.
func lastNonWhiteSpace(s []byte) byte {
	for i := len(s) - 1; i >= 0; i-- {
		if x := s[i]; (x != ' ') && (x != '\t') {
			return x
		}
	}
	return 0
}

// skipCooked converts `ijk \" lmn" pqr` to ` pqr`.
func skipCooked(s []byte, quote byte) (suffix []byte) {
	for i := 0; i < len(s); {
		if x := s[i]; x == quote {
			return s[i+1:]
		} else if x != '\\' {
			i += 1
		} else if (i + 1) < len(s) {
			i += 2
		} else {
			break
		}
	}
	return nil
}

// handleRaw copies a raw string from restOfSrc to dst, re-calculating the
// (line, remaining) pair afterwards.
func handleRaw(dst []byte, restOfSrc []byte, endQuote []byte) (retDst []byte, line []byte, remaining []byte) {
	end := bytes.Index(restOfSrc, endQuote)
	if end < 0 {
		end = len(restOfSrc)
	} else {
		end += len(endQuote)
	}
	dst = append(dst, restOfSrc[:end]...)
	line, remaining = restOfSrc[end:], nil
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		line, remaining = line[:i], line[i+1:]
	}
	return dst, line, remaining
}
