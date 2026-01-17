package base62

import (
	"math"
	"testing"
)

func TestEncode(t *testing.T) {
	testCases := []struct {
		input    uint64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "a"},
		{35, "z"},
		{36, "A"},
		{61, "Z"},
		{62, "10"},
		{12345, "3d7"},
		{123456789, "8m0Kx"},
		{1000000000, "15FTGg"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := Encode(tc.input)
			if result != tc.expected {
				t.Errorf("Encode(%d) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	testCases := []struct {
		input    string
		expected uint64
	}{
		{"0", 0},
		{"1", 1},
		{"9", 9},
		{"a", 10},
		{"z", 35},
		{"A", 36},
		{"Z", 61},
		{"10", 62},
		{"3d7", 12345},
		{"8m0Kx", 123456789},
		{"15FTGg", 1000000000},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := Decode(tc.input)
			if err != nil {
				t.Fatalf("Decode(%q) returned error: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("Decode(%q) = %d, want %d", tc.input, result, tc.expected)
			}
		})
	}
}

func TestDecodeInvalidCharacter(t *testing.T) {
	invalidInputs := []string{
		"",
		"!",
		"abc!",
		"hello world",
		"-1",
		"abc-def",
	}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			_, err := Decode(input)
			if err != ErrInvalidCharacter {
				t.Errorf("Decode(%q) expected ErrInvalidCharacter, got %v", input, err)
			}
		})
	}
}

func TestEncodeDecodeRoundtrip(t *testing.T) {
	testCases := []uint64{
		0,
		1,
		10,
		62,
		100,
		1000,
		12345,
		123456789,
		1000000000,
		math.MaxUint32,
		math.MaxUint64,
	}

	for _, num := range testCases {
		encoded := Encode(num)
		decoded, err := Decode(encoded)
		if err != nil {
			t.Fatalf("Decode(%q) returned error: %v", encoded, err)
		}
		if decoded != num {
			t.Errorf("Roundtrip failed: %d -> %q -> %d", num, encoded, decoded)
		}
	}
}

func TestMaxUint64(t *testing.T) {
	maxVal := uint64(math.MaxUint64)
	encoded := Encode(maxVal)
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode(%q) returned error: %v", encoded, err)
	}
	if decoded != maxVal {
		t.Errorf("MaxUint64 roundtrip failed: %d -> %q -> %d", maxVal, encoded, decoded)
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Encode(123456789)
	}
}

func BenchmarkDecode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Decode("8m0Kx")
	}
}

func BenchmarkEncodeMaxUint64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Encode(math.MaxUint64)
	}
}

func BenchmarkDecodeMaxUint64(b *testing.B) {
	encoded := Encode(math.MaxUint64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decode(encoded)
	}
}