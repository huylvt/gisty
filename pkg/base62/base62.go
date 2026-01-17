package base62

import (
	"errors"
	"strings"
)

// Charset contains the 62 characters used for encoding: 0-9, a-z, A-Z
const Charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const base = 62

var (
	// ErrInvalidCharacter is returned when the input contains invalid characters
	ErrInvalidCharacter = errors.New("base62: invalid character in input")
)

// charIndex maps each character to its index for fast decoding
var charIndex [256]int

func init() {
	// Initialize all indices to -1 (invalid)
	for i := range charIndex {
		charIndex[i] = -1
	}
	// Set valid character indices
	for i, c := range Charset {
		charIndex[c] = i
	}
}

// Encode converts a uint64 number to a base62 string
func Encode(num uint64) string {
	if num == 0 {
		return string(Charset[0])
	}

	var result strings.Builder
	result.Grow(11) // max length for uint64 in base62

	for num > 0 {
		remainder := num % base
		result.WriteByte(Charset[remainder])
		num /= base
	}

	// Reverse the string
	encoded := result.String()
	runes := []rune(encoded)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

// Decode converts a base62 string back to a uint64 number
func Decode(s string) (uint64, error) {
	if len(s) == 0 {
		return 0, ErrInvalidCharacter
	}

	var num uint64
	for _, c := range s {
		if c > 255 {
			return 0, ErrInvalidCharacter
		}
		idx := charIndex[c]
		if idx < 0 {
			return 0, ErrInvalidCharacter
		}
		num = num*base + uint64(idx)
	}

	return num, nil
}