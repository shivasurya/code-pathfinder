package graph

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"
)

func TestGenerateMethodID(t *testing.T) {
	tests := []struct {
		name       string
		methodName string
		parameters []string
		sourceFile string
		want       int
	}{
		{
			name:       "Simple method",
			methodName: "testMethod",
			parameters: []string{"int", "string"},
			sourceFile: "Test.java",
			want:       64,
		},
		{
			name:       "Empty parameters",
			methodName: "emptyParams",
			parameters: []string{},
			sourceFile: "Empty.java",
			want:       64,
		},
		{
			name:       "Long method name",
			methodName: "thisIsAVeryLongMethodNameThatExceedsTwentyCharacters",
			parameters: []string{"long"},
			sourceFile: "LongName.java",
			want:       64,
		},
		{
			name:       "Special characters",
			methodName: "special$Method#Name",
			parameters: []string{"char[]", "int[]"},
			sourceFile: "Special!File@Name.java",
			want:       64,
		},
		{
			name:       "Unicode characters",
			methodName: "unicodeMethod你好",
			parameters: []string{"String"},
			sourceFile: "Unicode文件.java",
			want:       64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateMethodID(tt.methodName, tt.parameters, tt.sourceFile)
			if len(got) != tt.want {
				t.Errorf("GenerateMethodID() returned ID with incorrect length, got %d, want %d", len(got), tt.want)
			}
			if !isValidHexString(got) {
				t.Errorf("GenerateMethodID() returned invalid hex string: %s", got)
			}
		})
	}
}

func TestGenerateSha256(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "Empty string",
			input: "",
			want:  64,
		},
		{
			name:  "Simple string",
			input: "Hello, World!",
			want:  64,
		},
		{
			name:  "Long string",
			input: "This is a very long string that exceeds sixty-four characters in length",
			want:  64,
		},
		{
			name:  "Special characters",
			input: "!@#$%^&*()_+{}[]|\\:;\"'<>,.?/",
			want:  64,
		},
		{
			name:  "Unicode characters",
			input: "こんにちは世界",
			want:  64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSha256(tt.input)
			if len(got) != tt.want {
				t.Errorf("GenerateSha256() returned hash with incorrect length, got %d, want %d", len(got), tt.want)
			}
			if !isValidHexString(got) {
				t.Errorf("GenerateSha256() returned invalid hex string: %s", got)
			}
		})
	}
}

func isValidHexString(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}

func TestFormatType(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{
			name:  "String input",
			input: "test string",
			want:  "test string",
		},
		{
			name:  "Integer input",
			input: 42,
			want:  "42",
		},
		{
			name:  "Int64 input",
			input: int64(9223372036854775807),
			want:  "9223372036854775807",
		},
		{
			name:  "Float32 input",
			input: float32(3.14),
			want:  "3.14",
		},
		{
			name:  "Float64 input",
			input: 2.71828,
			want:  "2.72",
		},
		{
			name:  "Slice of integers",
			input: []interface{}{1, 2, 3},
			want:  "[1,2,3]",
		},
		{
			name:  "Slice of mixed types",
			input: []interface{}{"a", 1, true},
			want:  `["a",1,true]`,
		},
		{
			name:  "Boolean input",
			input: true,
			want:  "true",
		},
		{
			name:  "Nil input",
			input: nil,
			want:  "<nil>",
		},
		{
			name:  "Struct input",
			input: struct{ Name string }{"John"},
			want:  "{John}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatType(tt.input)
			if got != tt.want {
				t.Errorf("FormatType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnableVerboseLogging(t *testing.T) {
	// Reset verboseFlag before test
	verboseFlag = false

	EnableVerboseLogging()

	if !verboseFlag {
		t.Error("EnableVerboseLogging() did not set verboseFlag to true")
	}
}

func TestLog(t *testing.T) {
	tests := []struct {
		name    string
		message string
		args    []interface{}
		verbose bool
	}{
		{
			name:    "Verbose logging enabled",
			message: "Test message",
			args:    []interface{}{1, "two", true},
			verbose: true,
		},
		{
			name:    "Verbose logging disabled",
			message: "Another test message",
			args:    []interface{}{3.14, []int{1, 2, 3}},
			verbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verboseFlag = tt.verbose

			// Redirect log output
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr)

			Log(tt.message, tt.args...)

			logOutput := buf.String()
			if tt.verbose {
				if !strings.Contains(logOutput, tt.message) {
					t.Errorf("Log() output does not contain expected message: %s", tt.message)
				}
				for _, arg := range tt.args {
					if !strings.Contains(logOutput, fmt.Sprint(arg)) {
						t.Errorf("Log() output does not contain expected argument: %v", arg)
					}
				}
			} else if logOutput != "" {
				t.Errorf("Log() produced output when verbose logging was disabled")
			}
		})
	}
}

func TestFmt(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		args    []interface{}
		verbose bool
		want    string
	}{
		{
			name:    "Verbose formatting enabled",
			format:  "Number: %d, String: %s, Float: %.2f",
			args:    []interface{}{42, "test", 3.14159},
			verbose: true,
			want:    "Number: 42, String: test, Float: 3.14",
		},
		{
			name:    "Verbose formatting disabled",
			format:  "This should not be printed: %v",
			args:    []interface{}{"ignored"},
			verbose: false,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verboseFlag = tt.verbose

			// Redirect stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			Fmt(tt.format, tt.args...)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			got := buf.String()

			if got != tt.want {
				t.Errorf("Fmt() output = %q, want %q", got, tt.want)
			}
		})
	}
}
