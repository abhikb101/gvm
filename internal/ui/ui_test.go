package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestStyleFunctions(t *testing.T) {
	// Force color on for tests
	colorEnabled = true

	tests := []struct {
		name string
		fn   func(string) string
		code string
	}{
		{"Bold", Bold, bold},
		{"Dim", Dim, dim},
		{"Red", Red, red},
		{"Green", Green, green},
		{"Yellow", Yellow, yellow},
		{"Blue", Blue, blue},
		{"Magenta", Magenta, magenta},
		{"Cyan", Cyan, cyan},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn("hello")
			if !strings.Contains(result, "hello") {
				t.Errorf("%s(\"hello\") = %q, missing text", tt.name, result)
			}
			if !strings.HasPrefix(result, tt.code) {
				t.Errorf("%s(\"hello\") missing ANSI code prefix", tt.name)
			}
			if !strings.HasSuffix(result, reset) {
				t.Errorf("%s(\"hello\") missing reset suffix", tt.name)
			}
		})
	}
}

func TestStyleDisabled(t *testing.T) {
	colorEnabled = false
	defer func() { colorEnabled = true }()

	result := Bold("hello")
	if result != "hello" {
		t.Errorf("Bold(\"hello\") with color disabled = %q, want \"hello\"", result)
	}

	result = Red("test")
	if result != "test" {
		t.Errorf("Red(\"test\") with color disabled = %q, want \"test\"", result)
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"\033[31mhello\033[0m", "hello"},
		{"\033[1m\033[32mtest\033[0m", "test"},
		{"no escape", "no escape"},
		{"", ""},
	}

	for _, tt := range tests {
		got := stripANSI(tt.input)
		if got != tt.want {
			t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTableRender(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	table := NewTable()
	table.AddRow("name", "email", "auth")
	table.AddRow("personal", "john@example.com", "ssh")
	table.AddRow("work", "john@company.com", "ssh+http")
	table.Render()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "personal") {
		t.Error("table output missing 'personal'")
	}
	if !strings.Contains(output, "john@example.com") {
		t.Error("table output missing email")
	}
	if !strings.Contains(output, "ssh+http") {
		t.Error("table output missing auth method")
	}
}

func TestTableAlignment(t *testing.T) {
	table := NewTable()
	table.AddRow("a", "b")
	table.AddRow("longer", "val")

	if len(table.rows) != 2 {
		t.Errorf("table has %d rows, want 2", len(table.rows))
	}
	if table.widths[0] != 6 { // "longer" = 6
		t.Errorf("column 0 width = %d, want 6", table.widths[0])
	}
}

func TestOutputFunctions(t *testing.T) {
	colorEnabled = true

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Success("test %s", "msg")
	Fail("error %s", "msg")
	Warn("warning %s", "msg")
	Info("info %s", "msg")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "test msg") {
		t.Error("Success() output missing message")
	}
	if !strings.Contains(output, "error msg") {
		t.Error("Fail() output missing message")
	}
	if !strings.Contains(output, "warning msg") {
		t.Error("Warn() output missing message")
	}
	if !strings.Contains(output, "info msg") {
		t.Error("Info() output missing message")
	}
}

func TestErrorf(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Errorf("test error %d", 42)

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "test error 42") {
		t.Errorf("Errorf() output = %q, missing message", output)
	}
}

func TestNewSpinnerNonTTY(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	s := NewSpinner("loading")
	s.Stop(true)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if buf.Len() == 0 {
		t.Error("spinner produced no output in non-TTY mode")
	}
}

func TestSpinnerStopIdempotent(t *testing.T) {
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	s := NewSpinner("test")
	s.Stop(true)
	s.Stop(false) // should not panic

	w.Close()
	os.Stdout = old
}

func TestSpinnerStopWithMessage(t *testing.T) {
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	s := NewSpinner("initial")
	s.StopWithMessage(true, "final message")

	w.Close()
	os.Stdout = old
}
