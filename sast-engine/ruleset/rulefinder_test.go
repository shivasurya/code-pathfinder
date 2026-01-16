package ruleset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleFinder_FindRuleFile(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create test directory structure
	dockerDir := filepath.Join(tmpDir, "docker")
	securityDir := filepath.Join(dockerDir, "security")
	bpDir := filepath.Join(dockerDir, "best-practice")

	require.NoError(t, os.MkdirAll(securityDir, 0755))
	require.NoError(t, os.MkdirAll(bpDir, 0755))

	// Create test Python files with rule IDs
	testFiles := map[string]string{
		filepath.Join(securityDir, "privileged_mode.py"): `@dockerfile_rule(
    id="DOCKER-SEC-001",
    name="Privileged Mode",
    severity="CRITICAL"
)
def check_privileged():
    pass
`,
		filepath.Join(bpDir, "apk_no_cache.py"): `@dockerfile_rule(
    id="DOCKER-BP-007",
    name="apk without --no-cache",
    severity="LOW"
)
def apk_no_cache():
    pass
`,
		filepath.Join(bpDir, "apt_recommends.py"): `@dockerfile_rule(
    id="DOCKER-BP-005",
    name="apt without --no-install-recommends",
    severity="LOW"
)
def apt_recommends():
    pass
`,
	}

	for path, content := range testFiles {
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	}

	finder := NewRuleFinder(tmpDir)

	tests := []struct {
		name     string
		spec     *RuleSpec
		wantFile string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "find DOCKER-SEC-001",
			spec:     &RuleSpec{Language: "docker", RuleID: "DOCKER-SEC-001"},
			wantFile: "privileged_mode.py",
			wantErr:  false,
		},
		{
			name:     "find DOCKER-BP-007",
			spec:     &RuleSpec{Language: "docker", RuleID: "DOCKER-BP-007"},
			wantFile: "apk_no_cache.py",
			wantErr:  false,
		},
		{
			name:     "find DOCKER-BP-005",
			spec:     &RuleSpec{Language: "docker", RuleID: "DOCKER-BP-005"},
			wantFile: "apt_recommends.py",
			wantErr:  false,
		},
		{
			name:    "rule not found",
			spec:    &RuleSpec{Language: "docker", RuleID: "DOCKER-BP-999"},
			wantErr: true,
			errMsg:  "rule DOCKER-BP-999 not found",
		},
		{
			name:    "language directory not found",
			spec:    &RuleSpec{Language: "python", RuleID: "PYTHON-SEC-001"},
			wantErr: true,
			errMsg:  "language directory not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := finder.FindRuleFile(tt.spec)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Contains(t, got, tt.wantFile)
				// Verify the file exists
				_, err := os.Stat(got)
				assert.NoError(t, err)
			}
		})
	}
}

func TestFileContainsRuleID(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		ruleID   string
		want     bool
		wantErr  bool
	}{
		{
			name: "rule ID with double quotes",
			content: `@dockerfile_rule(
    id="DOCKER-BP-007",
    name="Test"
)`,
			ruleID:  "DOCKER-BP-007",
			want:    true,
			wantErr: false,
		},
		{
			name: "rule ID with single quotes",
			content: `@dockerfile_rule(
    id='DOCKER-BP-007',
    name='Test'
)`,
			ruleID:  "DOCKER-BP-007",
			want:    true,
			wantErr: false,
		},
		{
			name: "rule ID not present",
			content: `@dockerfile_rule(
    id="DOCKER-BP-999",
    name="Test"
)`,
			ruleID:  "DOCKER-BP-007",
			want:    false,
			wantErr: false,
		},
		{
			name: "partial match should not match",
			content: `@dockerfile_rule(
    id="DOCKER-BP-0071",
    name="Test"
)`,
			ruleID:  "DOCKER-BP-007",
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := filepath.Join(tmpDir, "test.py")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.content), 0644))
			defer os.Remove(tmpFile)

			got, err := fileContainsRuleID(tmpFile, tt.ruleID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestRuleFinder_SkipsSpecialFiles(t *testing.T) {
	tmpDir := t.TempDir()
	dockerDir := filepath.Join(tmpDir, "docker")
	require.NoError(t, os.MkdirAll(dockerDir, 0755))

	// Create files that should be skipped
	skipFiles := []string{
		"__init__.py",
		"__pycache__.py",
	}

	for _, file := range skipFiles {
		content := `id="DOCKER-TEST-001"`
		require.NoError(t, os.WriteFile(filepath.Join(dockerDir, file), []byte(content), 0644))
	}

	// Create valid file
	validFile := filepath.Join(dockerDir, "test_rule.py")
	validContent := `@dockerfile_rule(
    id="DOCKER-TEST-002",
    name="Test"
)
def test_rule():
    pass
`
	require.NoError(t, os.WriteFile(validFile, []byte(validContent), 0644))

	finder := NewRuleFinder(tmpDir)

	// Should skip __init__.py and find the valid file
	spec := &RuleSpec{Language: "docker", RuleID: "DOCKER-TEST-002"}
	got, err := finder.FindRuleFile(spec)
	require.NoError(t, err)
	assert.Contains(t, got, "test_rule.py")
}
