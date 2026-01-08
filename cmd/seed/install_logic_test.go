package main

import (
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/krisarmstrong/seed/internal/paths"
)

func TestCopyFileLogic(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	// Create source file
	content := []byte("test content for copy")
	if err := os.WriteFile(srcPath, content, 0o644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	err := copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify destination exists and has correct content
	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(data) != string(content) {
		t.Errorf("Content mismatch: got %q, want %q", string(data), string(content))
	}
}

func TestCopyFileNonExistentSource(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "nonexistent.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	err := copyFile(srcPath, dstPath)
	if err == nil {
		t.Error("Expected error for non-existent source file")
	}
}

func TestCreateInstallDirectoriesLogic(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dirs := []string{
		filepath.Join(tmpDir, "config"),
		filepath.Join(tmpDir, "data"),
		filepath.Join(tmpDir, "logs"),
		filepath.Join(tmpDir, "cache"),
	}

	err := createInstallDirectories(dirs)
	if err != nil {
		t.Fatalf("createInstallDirectories failed: %v", err)
	}

	// Verify all directories were created
	for _, dir := range dirs {
		info, statErr := os.Stat(dir)
		if statErr != nil {
			t.Errorf("Directory %q was not created: %v", dir, statErr)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q should be a directory", dir)
		}
	}
}

func TestCreateInstallDirectoriesNestedPaths(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dirs := []string{
		filepath.Join(tmpDir, "a", "b", "c"),
		filepath.Join(tmpDir, "x", "y", "z"),
	}

	err := createInstallDirectories(dirs)
	if err != nil {
		t.Fatalf("createInstallDirectories failed with nested paths: %v", err)
	}

	for _, dir := range dirs {
		if _, statErr := os.Stat(dir); statErr != nil {
			t.Errorf("Nested directory %q was not created: %v", dir, statErr)
		}
	}
}

func TestResolveBinaryDestinationUser(t *testing.T) {
	t.Parallel()

	// Test user mode
	p := &paths.Paths{
		BinaryDir: "/usr/local/bin",
	}

	dest, err := resolveBinaryDestination(paths.ModeUser, p)
	if err != nil {
		t.Fatalf("resolveBinaryDestination failed: %v", err)
	}

	// User mode should use ~/.local/bin
	if !containsSubstring(dest, ".local/bin/seed") {
		t.Errorf("User mode destination should contain .local/bin/seed, got: %s", dest)
	}
}

func TestResolveBinaryDestinationSystem(t *testing.T) {
	t.Parallel()

	p := &paths.Paths{
		BinaryDir: "/usr/local/bin",
	}

	dest, err := resolveBinaryDestination(paths.ModeSystem, p)
	if err != nil {
		t.Fatalf("resolveBinaryDestination failed: %v", err)
	}

	expected := filepath.Join(p.BinaryDir, "seed")
	if dest != expected {
		t.Errorf("System mode destination: got %q, want %q", dest, expected)
	}
}

func TestInstallBinaryWithForce(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source")
	destPath := filepath.Join(tmpDir, "dest")

	// Create source file
	if err := os.WriteFile(srcPath, []byte("source content"), 0o755); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	// Create existing destination
	if err := os.WriteFile(destPath, []byte("old content"), 0o755); err != nil {
		t.Fatalf("Failed to create existing dest: %v", err)
	}

	// Test that without force, it doesn't overwrite
	err := installBinary(srcPath, destPath, false)
	if err != nil {
		t.Fatalf("installBinary failed: %v", err)
	}

	// Content should still be old
	data, _ := os.ReadFile(destPath)
	if string(data) != "old content" {
		t.Error("Without force, binary should not be overwritten")
	}

	// Now test with force
	err = installBinary(srcPath, destPath, true)
	if err != nil {
		t.Fatalf("installBinary with force failed: %v", err)
	}

	// Content should now be new
	data, _ = os.ReadFile(destPath)
	if string(data) != "source content" {
		t.Error("With force, binary should be overwritten")
	}
}

func TestInstallBinaryNewDestination(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source")
	destPath := filepath.Join(tmpDir, "dest")

	// Create source file
	if err := os.WriteFile(srcPath, []byte("new binary"), 0o755); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	// Install to new location
	err := installBinary(srcPath, destPath, false)
	if err != nil {
		t.Fatalf("installBinary failed: %v", err)
	}

	// Verify destination was created
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read destination: %v", err)
	}

	if string(data) != "new binary" {
		t.Errorf("Destination content mismatch: got %q, want %q", string(data), "new binary")
	}
}

func TestServiceConfigTemplate(t *testing.T) {
	t.Parallel()

	cfg := serviceConfig{
		User:       "seed",
		Group:      "seed",
		BinaryPath: "/usr/local/bin/seed",
		ConfigDir:  "/etc/seed",
		DataDir:    "/var/lib/seed",
		LogDir:     "/var/log/seed",
		CacheDir:   "/var/cache/seed",
	}

	// Test that the template can be parsed and executed
	tmpl, err := template.New("test").Parse(systemdServiceTemplate)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	var buf []byte
	buf = make([]byte, 0, 4096)
	writer := &byteSliceWriter{buf: &buf}

	err = tmpl.Execute(writer, cfg)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	output := string(buf)

	// Verify placeholders were replaced
	expectedContent := []string{
		"seed", // User
		"/usr/local/bin/seed",
		"/etc/seed",
		"/var/lib/seed",
		"/var/log/seed",
	}

	for _, content := range expectedContent {
		if !containsSubstring(output, content) {
			t.Errorf("Template output should contain %q", content)
		}
	}
}

func TestUserServiceConfigTemplate(t *testing.T) {
	t.Parallel()

	cfg := serviceConfig{
		BinaryPath: "/home/user/.local/bin/seed",
	}

	// Test that the user template can be parsed and executed
	tmpl, err := template.New("test").Parse(userServiceTemplate)
	if err != nil {
		t.Fatalf("Failed to parse user template: %v", err)
	}

	var buf []byte
	buf = make([]byte, 0, 4096)
	writer := &byteSliceWriter{buf: &buf}

	err = tmpl.Execute(writer, cfg)
	if err != nil {
		t.Fatalf("Failed to execute user template: %v", err)
	}

	output := string(buf)

	// Verify binary path was replaced
	if !containsSubstring(output, "/home/user/.local/bin/seed") {
		t.Error("User template output should contain binary path")
	}

	// Verify it uses default.target
	if !containsSubstring(output, "default.target") {
		t.Error("User template should use default.target")
	}
}

func TestPrintCompletionMessageDoesNotPanic(t *testing.T) {
	t.Parallel()

	// Test that printCompletionMessage doesn't panic for either mode
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printCompletionMessage panicked: %v", r)
		}
	}()

	// We can't capture stdout easily in parallel tests,
	// so just verify it doesn't panic
	printCompletionMessage(paths.ModeSystem)
	printCompletionMessage(paths.ModeUser)
}

func TestCreateDefaultConfigLogic(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Call createDefaultConfig for a new directory
	createDefaultConfig(tmpDir)

	// Verify config file was created
	configPath := filepath.Join(tmpDir, "seed.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("Config file should be created: %v", err)
	}
}

func TestCreateDefaultConfigExisting(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "seed.yaml")

	// Create existing config with specific content
	originalContent := []byte("# existing config\nversion: 1\n")
	if err := os.WriteFile(configPath, originalContent, 0o600); err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	// Call createDefaultConfig
	createDefaultConfig(tmpDir)

	// Verify existing config was NOT overwritten
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if string(data) != string(originalContent) {
		t.Error("Existing config should not be overwritten")
	}
}

// byteSliceWriter is a simple io.Writer for testing templates.
type byteSliceWriter struct {
	buf *[]byte
}

func (w *byteSliceWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}
