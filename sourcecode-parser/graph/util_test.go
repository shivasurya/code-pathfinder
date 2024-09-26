package graph

import (
	"encoding/hex"
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
