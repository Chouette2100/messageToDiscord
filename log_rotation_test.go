package main

import (
	"testing"
	"time"
)

func TestParseRotateAt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantH   int
		wantM   int
		wantErr bool
	}{
		{name: "default empty", input: "", wantH: 0, wantM: 0},
		{name: "trimmed", input: " 06:30 ", wantH: 6, wantM: 30},
		{name: "invalid format", input: "0630", wantErr: true},
		{name: "invalid hour", input: "24:00", wantErr: true},
		{name: "invalid minute", input: "12:60", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m, err := parseRotateAt(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseRotateAt(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRotateAt(%q) error = %v", tt.input, err)
			}
			if h != tt.wantH || m != tt.wantM {
				t.Fatalf("parseRotateAt(%q) = (%d,%d), want (%d,%d)", tt.input, h, m, tt.wantH, tt.wantM)
			}
		})
	}
}

func TestNextRotationTime(t *testing.T) {
	loc := time.FixedZone("JST", 9*60*60)

	t.Run("same day", func(t *testing.T) {
		now := time.Date(2026, 10, 28, 8, 0, 0, 0, loc)
		got := nextRotationTime(now, 10, 30)
		want := time.Date(2026, 10, 28, 10, 30, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("nextRotationTime() = %v, want %v", got, want)
		}
	})

	t.Run("next day when same time", func(t *testing.T) {
		now := time.Date(2026, 10, 28, 10, 30, 0, 0, loc)
		got := nextRotationTime(now, 10, 30)
		want := time.Date(2026, 10, 29, 10, 30, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("nextRotationTime() = %v, want %v", got, want)
		}
	})

	t.Run("next day when passed", func(t *testing.T) {
		now := time.Date(2026, 10, 28, 23, 0, 0, 0, loc)
		got := nextRotationTime(now, 22, 0)
		want := time.Date(2026, 10, 29, 22, 0, 0, 0, loc)
		if !got.Equal(want) {
			t.Fatalf("nextRotationTime() = %v, want %v", got, want)
		}
	})
}

func TestShouldRotateNow(t *testing.T) {
	loc := time.FixedZone("JST", 9*60*60)
	scheduled := time.Date(2026, 10, 29, 0, 0, 0, 0, loc)

	t.Run("before scheduled", func(t *testing.T) {
		now := time.Date(2026, 10, 28, 23, 59, 59, 0, loc)
		if shouldRotateNow(now, scheduled) {
			t.Fatalf("shouldRotateNow() = true, want false")
		}
	})

	t.Run("at scheduled", func(t *testing.T) {
		now := time.Date(2026, 10, 29, 0, 0, 0, 0, loc)
		if !shouldRotateNow(now, scheduled) {
			t.Fatalf("shouldRotateNow() = false, want true")
		}
	})

	t.Run("after scheduled", func(t *testing.T) {
		now := time.Date(2026, 10, 29, 0, 0, 1, 0, loc)
		if !shouldRotateNow(now, scheduled) {
			t.Fatalf("shouldRotateNow() = false, want true")
		}
	})
}
