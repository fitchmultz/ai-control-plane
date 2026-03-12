// colors_test.go - Coverage for output styling and status-symbol helpers.
//
// Purpose:
//   - Verify the output helpers preserve byte-for-byte styling behavior while
//     remaining deterministic in color and no-color modes.
//
// Responsibilities:
//   - Cover text styling helpers for ANSI-wrapped and plain-text output.
//   - Cover symbol helpers with and without trailing status text.
//
// Scope:
//   - internal/output helper behavior only.
//
// Usage:
//   - Run via `go test ./internal/output`.
//
// Invariants/Assumptions:
//   - Force-color mode always wraps with the expected ANSI prefix/reset bytes.
//   - No-color mode strips styling while keeping text and symbols unchanged.
package output

import "testing"

func TestStyledTextHelpers(t *testing.T) {
	t.Parallel()

	force := NewForceColor()
	noColor := NewNoColor()

	tests := []struct {
		name       string
		forceGot   string
		noColorGot string
		forceWant  string
		plainWant  string
	}{
		{
			name:       "red",
			forceGot:   force.Red("demo"),
			noColorGot: noColor.Red("demo"),
			forceWant:  colorRed + "demo" + colorReset,
			plainWant:  "demo",
		},
		{
			name:       "green",
			forceGot:   force.Green("demo"),
			noColorGot: noColor.Green("demo"),
			forceWant:  colorGreen + "demo" + colorReset,
			plainWant:  "demo",
		},
		{
			name:       "yellow",
			forceGot:   force.Yellow("demo"),
			noColorGot: noColor.Yellow("demo"),
			forceWant:  colorYellow + "demo" + colorReset,
			plainWant:  "demo",
		},
		{
			name:       "blue",
			forceGot:   force.Blue("demo"),
			noColorGot: noColor.Blue("demo"),
			forceWant:  colorBlue + "demo" + colorReset,
			plainWant:  "demo",
		},
		{
			name:       "bold",
			forceGot:   force.Bold("demo"),
			noColorGot: noColor.Bold("demo"),
			forceWant:  colorBold + "demo" + colorReset,
			plainWant:  "demo",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.forceGot != tt.forceWant {
				t.Fatalf("force-color output = %q, want %q", tt.forceGot, tt.forceWant)
			}
			if tt.noColorGot != tt.plainWant {
				t.Fatalf("no-color output = %q, want %q", tt.noColorGot, tt.plainWant)
			}
		})
	}
}

func TestStatusSymbolHelpers(t *testing.T) {
	t.Parallel()

	force := NewForceColor()
	noColor := NewNoColor()

	tests := []struct {
		name       string
		emptyGot   string
		textGot    string
		emptyWant  string
		textWant   string
		plainEmpty string
		plainText  string
	}{
		{
			name:       "pass",
			emptyGot:   force.Pass(""),
			textGot:    force.Pass("ready"),
			emptyWant:  colorGreen + symbolPass + colorReset,
			textWant:   colorGreen + symbolPass + colorReset + " ready",
			plainEmpty: symbolPass,
			plainText:  symbolPass + " ready",
		},
		{
			name:       "fail",
			emptyGot:   force.Fail(""),
			textGot:    force.Fail("boom"),
			emptyWant:  colorRed + symbolFail + colorReset,
			textWant:   colorRed + symbolFail + colorReset + " boom",
			plainEmpty: symbolFail,
			plainText:  symbolFail + " boom",
		},
		{
			name:       "warn",
			emptyGot:   force.Warn(""),
			textGot:    force.Warn("careful"),
			emptyWant:  colorYellow + symbolWarn + colorReset,
			textWant:   colorYellow + symbolWarn + colorReset + " careful",
			plainEmpty: symbolWarn,
			plainText:  symbolWarn + " careful",
		},
		{
			name:       "info",
			emptyGot:   force.Info(""),
			textGot:    force.Info("heads up"),
			emptyWant:  colorBlue + symbolInfo + colorReset,
			textWant:   colorBlue + symbolInfo + colorReset + " heads up",
			plainEmpty: symbolInfo,
			plainText:  symbolInfo + " heads up",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.emptyGot != tt.emptyWant {
				t.Fatalf("force-color empty output = %q, want %q", tt.emptyGot, tt.emptyWant)
			}
			if tt.textGot != tt.textWant {
				t.Fatalf("force-color text output = %q, want %q", tt.textGot, tt.textWant)
			}

			var plainEmpty, plainText string
			switch tt.name {
			case "pass":
				plainEmpty = noColor.Pass("")
				plainText = noColor.Pass("ready")
			case "fail":
				plainEmpty = noColor.Fail("")
				plainText = noColor.Fail("boom")
			case "warn":
				plainEmpty = noColor.Warn("")
				plainText = noColor.Warn("careful")
			case "info":
				plainEmpty = noColor.Info("")
				plainText = noColor.Info("heads up")
			}

			if plainEmpty != tt.plainEmpty {
				t.Fatalf("no-color empty output = %q, want %q", plainEmpty, tt.plainEmpty)
			}
			if plainText != tt.plainText {
				t.Fatalf("no-color text output = %q, want %q", plainText, tt.plainText)
			}
		})
	}
}
