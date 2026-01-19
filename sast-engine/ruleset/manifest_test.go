package ruleset

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewManifestLoader(t *testing.T) {
	baseURL := "https://example.com"
	cacheDir := "/tmp/cache"

	loader := NewManifestLoader(baseURL, cacheDir)

	if loader.baseURL != baseURL {
		t.Errorf("expected baseURL %s, got %s", baseURL, loader.baseURL)
	}

	if loader.cacheDir != cacheDir {
		t.Errorf("expected cacheDir %s, got %s", cacheDir, loader.cacheDir)
	}

	if loader.httpClient == nil {
		t.Errorf("httpClient should not be nil")
	}
}

func TestLoadCategoryManifest(t *testing.T) {
	// Create test manifest
	testManifest := &Manifest{
		Category: "docker",
		Bundles: map[string]*Bundle{
			"security": {
				Name:        "Docker Security",
				Description: "Security rules",
				FileCount:   5,
				Checksum:    "abc123",
			},
		},
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/docker/manifest.json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testManifest)
	}))
	defer server.Close()

	loader := NewManifestLoader(server.URL, "")
	manifest, err := loader.LoadCategoryManifest("docker")
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	if manifest.Category != "docker" {
		t.Errorf("expected category docker, got %s", manifest.Category)
	}

	bundle, exists := manifest.Bundles["security"]
	if !exists {
		t.Fatalf("expected security bundle to exist")
	}

	if bundle.Name != "Docker Security" {
		t.Errorf("expected bundle name 'Docker Security', got %s", bundle.Name)
	}
}

func TestLoadCategoryManifest404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	loader := NewManifestLoader(server.URL, "")
	_, err := loader.LoadCategoryManifest("nonexistent")
	if err == nil {
		t.Errorf("expected error for 404, got nil")
	}
}

func TestManifestGetBundle(t *testing.T) {
	manifest := &Manifest{
		Bundles: map[string]*Bundle{
			"security": {
				Name: "Security Rules",
			},
			"best-practice": {
				Name: "Best Practices",
			},
		},
	}

	// Test existing bundle
	bundle, err := manifest.GetBundle("security")
	if err != nil {
		t.Fatalf("failed to get bundle: %v", err)
	}

	if bundle.Name != "Security Rules" {
		t.Errorf("expected name 'Security Rules', got %s", bundle.Name)
	}

	// Test non-existent bundle
	_, err = manifest.GetBundle("nonexistent")
	if err == nil {
		t.Errorf("expected error for non-existent bundle, got nil")
	}
}

func TestManifestGetAllBundleNames(t *testing.T) {
	tests := []struct {
		name     string
		manifest *Manifest
		want     []string
	}{
		{
			name: "multiple bundles",
			manifest: &Manifest{
				Bundles: map[string]*Bundle{
					"security":      {Name: "Security Rules"},
					"best-practice": {Name: "Best Practices"},
					"performance":   {Name: "Performance Rules"},
				},
			},
			want: []string{"best-practice", "performance", "security"},
		},
		{
			name: "single bundle",
			manifest: &Manifest{
				Bundles: map[string]*Bundle{
					"security": {Name: "Security Rules"},
				},
			},
			want: []string{"security"},
		},
		{
			name: "empty bundles",
			manifest: &Manifest{
				Bundles: map[string]*Bundle{},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.manifest.GetAllBundleNames()

			if len(got) != len(tt.want) {
				t.Errorf("expected %d bundles, got %d", len(tt.want), len(got))
				return
			}

			// Check each expected bundle name
			for i, name := range tt.want {
				if got[i] != name {
					t.Errorf("expected bundle[%d] = %s, got %s", i, name, got[i])
				}
			}
		})
	}
}
