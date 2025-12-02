package theme

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ThemeManifest represents a theme's configuration
type ThemeManifest struct {
	Name        string        `yaml:"name"`
	Version     string        `yaml:"version"`
	Author      string        `yaml:"author"`
	Description string        `yaml:"description"`
	License     string        `yaml:"license"`
	Features    ThemeFeatures `yaml:"features"`
	Styles      []string      `yaml:"styles"`
	Scripts     []string      `yaml:"scripts"`
}

// ThemeFeatures defines optional theme capabilities
type ThemeFeatures struct {
	Particles  bool `yaml:"particles"`
	Animations bool `yaml:"animations"`
}

// Load reads a theme manifest from the given theme directory.
// If no theme.yaml exists, it returns a default manifest for backward compatibility.
func Load(themePath string) (*ThemeManifest, error) {
	manifestPath := filepath.Join(themePath, "theme.yaml")

	// Check if manifest exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return legacyManifest(themePath), nil
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest ThemeManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	// Apply defaults for missing values
	applyDefaults(&manifest, themePath)

	return &manifest, nil
}

// legacyManifest creates a default manifest for themes without theme.yaml
func legacyManifest(themePath string) *ThemeManifest {
	manifest := &ThemeManifest{
		Name:     filepath.Base(themePath),
		Version:  "1.0.0",
		Features: ThemeFeatures{},
	}

	// Check for legacy styles.css at root level
	if _, err := os.Stat(filepath.Join(themePath, "styles.css")); err == nil {
		manifest.Styles = []string{"styles.css"}
	}

	return manifest
}

// applyDefaults fills in missing manifest values
func applyDefaults(manifest *ThemeManifest, themePath string) {
	if manifest.Name == "" {
		manifest.Name = filepath.Base(themePath)
	}
	if manifest.Version == "" {
		manifest.Version = "1.0.0"
	}

	// If no styles specified, check for common defaults
	if len(manifest.Styles) == 0 {
		// Check for styles directory
		if _, err := os.Stat(filepath.Join(themePath, "styles", "base.css")); err == nil {
			manifest.Styles = []string{"styles/base.css"}
		} else if _, err := os.Stat(filepath.Join(themePath, "styles.css")); err == nil {
			// Fall back to root-level styles.css
			manifest.Styles = []string{"styles.css"}
		}
	}
}
