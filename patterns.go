package router

import (
	"unicode"
)

type matchFunc func(string) bool

var functionMatchers = map[string]matchFunc{
	`isLowerAlpha`: isLowerAlpha,
	`isUpperAlpha`: isUpperAlpha,
	`isAlpha`:      isAlpha,
	`isDigits`:     isDigits,
	`isAlnum`:      isAlnum,
	`isWord`:       isWord,
	`isSlugSafe`:   isSlugSafe,
	`isSlug`:       isSlug,
	`isHex`:        isHex,
	`isUUID`:       isUUID,
	`isSafeText`:   isSafeText,
	`isUpperAlnum`: isUpperAlnum,
	`isBase64`:     isBase64,
	`isDateYMD`:    isDateYMD,
	`isSafePath`:   isSafePath,
	`any`:          isAny,
}

var patternMatchers = map[string]matchFunc{
	`[a-z]+`:            isLowerAlpha,
	`[A-Z]+`:            isUpperAlpha,
	`[a-zA-Z]+`:         isAlpha,
	`[0-9]+`:            isDigits,
	`\d+`:               isDigits,
	`[a-zA-Z0-9]+`:      isAlnum,
	`\w+`:               isWord,
	`[\w\-]+`:           isSlugSafe,
	`[a-z0-9\-]+`:       isSlug,
	`[a-fA-F0-9]+`:      isHex,
	`8-4-4-4-12`:        isUUID,
	`[a-zA-Z0-9 _.-]+`:  isSafeText,
	`[A-Z0-9]+`:         isUpperAlnum,
	`a-zA-Z0-9+/=`:      isBase64,
	`\d{4}-\d{2}-\d{2}`: isDateYMD,
	`[a-zA-Z0-9/._-]+`:  isSafePath,
	`.*`:                alwaysTrue,
}

func isAny(string) bool {
	return true
}

func isLowerAlpha(s string) bool {
	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		if s[i] < 'a' || s[i] > 'z' {
			return false
		}
	}

	return true
}

func isUpperAlpha(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 'A' || s[i] > 'Z' {
			return false
		}
	}
	return true
}

func isAlpha(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z')) {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

var isAlnumTable = [256]bool{
	'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true,
	'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true, 'H': true, 'I': true, 'J': true,
	'K': true, 'L': true, 'M': true, 'N': true, 'O': true, 'P': true, 'Q': true, 'R': true, 'S': true, 'T': true,
	'U': true, 'V': true, 'W': true, 'X': true, 'Y': true, 'Z': true,
	'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true, 'h': true, 'i': true, 'j': true,
	'k': true, 'l': true, 'm': true, 'n': true, 'o': true, 'p': true, 'q': true, 'r': true, 's': true, 't': true,
	'u': true, 'v': true, 'w': true, 'x': true, 'y': true, 'z': true,
}

func isAlnum(s string) bool {
	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		if !isAlnumTable[s[i]] {
			return false
		}
	}
	return true
}

func isWord(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if isAlnumTable[s[i]] || s[i] == '_' {
			continue
		}
		return false
	}
	return true
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_'
}

func isSlugSafe(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func isSlug(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= '0' && s[i] <= '9') || s[i] == '-' {
			continue
		}
		return false
	}
	return true
}

func isSpaceOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func alwaysTrue(string) bool {
	return true
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !(s[i] >= '0' && s[i] <= '9' || s[i] >= 'a' && s[i] <= 'f' || s[i] >= 'A' && s[i] <= 'F') {
			return false
		}
	}
	return true
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < 36; i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if s[i] != '-' {
				return false
			}
			continue
		}

		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

func isSafeText(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == ' ' || c == '_' || c == '.' || c == '-') {
			return false
		}
	}
	return true
}

func isUpperAlnum(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !(s[i] >= 'A' && s[i] <= 'Z' || s[i] >= '0' && s[i] <= '9') {
			return false
		}
	}
	return true
}

func isBase64(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !(s[i] >= 'a' && s[i] <= 'z' ||
			s[i] >= 'A' && s[i] <= 'Z' ||
			s[i] >= '0' && s[i] <= '9' ||
			s[i] == '+' || s[i] == '/' || s[i] == '=') {
			return false
		}
	}
	return true
}

func isDateYMD(s string) bool {
	if len(s) != 10 {
		return false
	}
	if s[4] != '-' || s[7] != '-' {
		return false
	}
	for i := 0; i < len(s); i++ {
		if i == 4 || i == 7 {
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isSafePath(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !(s[i] >= 'a' && s[i] <= 'z' ||
			s[i] >= 'A' && s[i] <= 'Z' ||
			s[i] >= '0' && s[i] <= '9' ||
			s[i] == '/' || s[i] == '.' || s[i] == '_' || s[i] == '-') {
			return false
		}
	}
	return true
}

func isValidURLSegment(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] >= 'a' && s[i] <= 'z':
		case s[i] >= 'A' && s[i] <= 'Z':
		case s[i] >= '0' && s[i] <= '9':
		case s[i] == '-' || s[i] == '_' || s[i] == '.':
		default:
			return false
		}
	}
	return true
}
