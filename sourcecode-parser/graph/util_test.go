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
