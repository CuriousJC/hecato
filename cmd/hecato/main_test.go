package main

import (
	"strings"
	"testing"
)

func TestPlural(t *testing.T) {
	tests := []struct {
		n    int
		one  string
		many string
		want string
	}{
		{0, "file", "files", "0 files"},
		{1, "file", "files", "1 file"},
		{2, "file", "files", "2 files"},
		{1, "directory", "directories", "1 directory"},
		{3, "directory", "directories", "3 directories"},
		{1, "path", "paths", "1 path"},
		{1000, "file", "files", "1,000 files"},
		{68361, "file", "files", "68,361 files"},
	}

	for _, tt := range tests {
		if got := plural(tt.n, tt.one, tt.many); got != tt.want {
			t.Errorf("plural(%d, %q, %q) = %q, want %q", tt.n, tt.one, tt.many, got, tt.want)
		}
	}
}

func TestComma(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1,000"},
		{1001, "1,001"},
		{9999, "9,999"},
		{10000, "10,000"},
		{100000, "100,000"},
		{999999, "999,999"},
		{1000000, "1,000,000"},
		{68361, "68,361"},
		{-1234, "-1,234"},
	}

	for _, tt := range tests {
		if got := comma(tt.n); got != tt.want {
			t.Errorf("comma(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

// The two method sets have to agree: anything that walks the filesystem must
// also be dispatchable, or a target check would guard a method doWork rejects.
func TestWalkingMethodsAreAllKnown(t *testing.T) {
	for m := range walkingMethods {
		if !knownMethods[m] {
			t.Errorf("%q walks the filesystem but is not in knownMethods", m)
		}
	}
}

func TestBar(t *testing.T) {
	const w = 8

	tests := []struct {
		name          string
		size, largest int64
		want          string
	}{
		{"largest fills the bar", 100, 100, "████████"},
		{"half", 50, 100, "████    "},
		{"quarter", 25, 100, "██      "},
		{"rounds up past the halfway point", 44, 100, "████    "},
		{"rounds down below it", 30, 100, "██      "},
		{"below half a cell is blank", 1, 1000, "        "},
		{"zero size is blank", 0, 100, "        "},
		{"zero largest is blank", 50, 0, "        "},
		{"negative largest is blank", 50, -1, "        "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bar(tt.size, tt.largest, w)
			if got != tt.want {
				t.Errorf("bar(%d, %d, %d) = %q, want %q", tt.size, tt.largest, w, got, tt.want)
			}
		})
	}
}

// Every bar must occupy the same number of cells, or the path column goes
// ragged. Partial blocks count as one cell, so the check is on runes.
func TestBarIsAlwaysTheSameWidth(t *testing.T) {
	const w = 16

	for _, size := range []int64{0, 1, 7, 100, 999, 12345, 1 << 40} {
		got := bar(size, 1<<40, w)
		if n := len([]rune(got)); n != w {
			t.Errorf("bar(%d, 1<<40, %d) is %d runes, want %d: %q", size, w, n, w, got)
		}
	}
}

// The bar must use only U+2588. The partial blocks U+2589-U+258F are missing
// from the Windows console font and render as tofu, so a bar built from them
// looks broken rather than coarse.
func TestBarUsesOnlyFullBlocks(t *testing.T) {
	for _, size := range []int64{1, 7, 33, 50, 99, 100} {
		got := bar(size, 100, 16)
		for _, r := range got {
			if r != ' ' && r != '█' {
				t.Errorf("bar(%d, 100, 16) contains %q (U+%04X); only U+2588 and space are safe",
					size, r, r)
			}
		}
	}
	_ = strings.TrimSpace
}
