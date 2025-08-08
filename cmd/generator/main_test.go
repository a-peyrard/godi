package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func findScriptPath() string {
	initialWd, _ := os.Getwd()
	if filepath.Base(initialWd) == "generator" {
		return filepath.Join(initialWd, "main.go")
	}
	return filepath.Join(initialWd, "cmd", "generator", "main.go")
}

func TestCodeGeneration(t *testing.T) {
	scriptPath := findScriptPath()

	testCases := []struct {
		name    string
		fixture string // directory name in etc/gen/
	}{
		{
			name:    "simple provider",
			fixture: "simple_provider",
		},
		{
			name:    "it should allow multi-lines description in providers",
			fixture: "multi_lines_description",
		},
		{
			name:    "provider with dependencies",
			fixture: "provider_with_deps",
		},
		{
			name:    "decorator",
			fixture: "decorator",
		},
		{
			name:    "config struct",
			fixture: "config",
		},
		{
			name:    "provider with conditions",
			fixture: "conditional_provider",
		},
		{
			name:    "multiple providers same name",
			fixture: "multiple_providers",
		},
		{
			name:    "complex scenario",
			fixture: "complex",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// GIVEN
			tempDir := setupTestProject(t, tc.fixture)

			// WHEN
			err := runGenerator(t, scriptPath, tempDir)

			// THEN
			require.NoError(t, err)
			assertGeneratedCode(t, tempDir, tc.fixture)
		})
	}
}

func setupTestProject(t *testing.T, fixture string) string {
	tempDir := t.TempDir()

	// Copy fixture files to temp directory
	fixtureDir := filepath.Join("etc", "gen", fixture)
	err := copyDir(fixtureDir, tempDir)
	require.NoError(t, err, "Failed to copy fixture files")

	return tempDir
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip golden files when copying
		if strings.HasSuffix(path, ".golden") {
			return nil
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}

func runGenerator(t *testing.T, scriptPath string, projectDir string) error {
	// Find the registry file
	registryFile := "registry.go"
	registryPath := filepath.Join(projectDir, registryFile)

	// Check if registry file exists in subdirectory for complex scenarios
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		// Try to find it in subdirectories
		err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Name() == "registry.go" {
				registryPath = path
				registryFile = info.Name()
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	registryDir := filepath.Dir(registryPath)
	registryPackage := getPackageName(t, registryPath)

	// Save current directory
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	// Build the generator binary first (from the module root where dependencies are available)
	generatorDir := filepath.Dir(scriptPath)
	generatorBinary := filepath.Join(t.TempDir(), "generator")

	buildCmd := exec.Command("go", "build", "-o", generatorBinary, ".")
	buildCmd.Dir = generatorDir

	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Logf("Failed to build generator:\n%s", buildOutput)
		return err
	}

	// Now run the built binary in the test directory
	cmd := exec.Command(generatorBinary)
	cmd.Dir = registryDir
	cmd.Env = append(os.Environ(),
		"GOFILE="+registryFile,
		"GOPACKAGE="+registryPackage,
		"DRY_RUN=false",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Generator output:\n%s", output)
		return err
	}

	return nil
}

func getPackageName(t *testing.T, goFile string) string {
	data, err := os.ReadFile(goFile)
	require.NoError(t, err)

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}

	t.Fatalf("Could not find package name in %s", goFile)
	return ""
}

func assertGeneratedCode(t *testing.T, projectDir string, fixture string) {
	// Find the generated file
	var generatedFile string
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), "_gen.go") {
			generatedFile = path
			return filepath.SkipDir
		}
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, generatedFile, "Generated file not found")

	actual, err := os.ReadFile(generatedFile)
	require.NoError(t, err)

	goldenFile := filepath.Join("etc", "gen", fixture, "expected_gen.go.golden")

	if *updateGolden {
		err = os.WriteFile(goldenFile, actual, 0644)
		require.NoError(t, err, "Failed to update golden file")
		t.Logf("Updated golden file: %s", goldenFile)
		return
	}

	expected, err := os.ReadFile(goldenFile)
	if os.IsNotExist(err) {
		t.Errorf("Golden file does not exist: %s\nRun with -update flag to create it", goldenFile)
		t.Logf("Generated content:\n%s", actual)
		return
	}
	require.NoError(t, err)

	assert.Equal(t, normalizeCode(string(expected)), normalizeCode(string(actual)))
}

func normalizeCode(code string) string {
	// Normalize line endings and trim trailing whitespace
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	return strings.Join(lines, "\n")
}
