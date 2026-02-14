package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateDirectory(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{"valid directory", "test-dir", false},
		{"nested directory", "test-dir/nested/subdir", false},
		{"existing directory", ".", false}, // Current directory should exist
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up after test
			defer func() {
				if tt.path != "." {
					if err := os.RemoveAll(tt.path); err != nil {
						t.Logf("Warning: failed to remove test directory: %v", err)
					}
				}
			}()

			err := CreateDirectory(tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("CreateDirectory(%s) expected error but got nil", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("CreateDirectory(%s) expected no error but got: %v", tt.path, err)
				}
				// Verify directory was created
				if tt.path != "." {
					if !DirectoryExists(tt.path) {
						t.Errorf("CreateDirectory(%s) should have created directory", tt.path)
					}
				}
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test-file-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Warning: failed to remove temp file: %v", err)
		}
	}()
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing file", tmpFile.Name(), true},
		{"non-existing file", "non-existing-file.txt", false},
		{"non-existing path", "non/existing/path.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FileExists(tt.path)
			if result != tt.expected {
				t.Errorf("FileExists(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestDirectoryExists(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test-dir-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-file-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Warning: failed to remove temp file: %v", err)
		}
	}()
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing directory", tmpDir, true},
		{"existing file (not directory)", tmpFile.Name(), false},
		{"non-existing path", "non/existing/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DirectoryExists(tt.path)
			if result != tt.expected {
				t.Errorf("DirectoryExists(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestWriteFile(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		content     string
		expectError bool
	}{
		{"simple file", "test-write.txt", "Hello, World!", false},
		{"file with newlines", "test-write-nl.txt", "Line 1\nLine 2\nLine 3", false},
		{"file in nested directory", "nested/test-write.txt", "Nested content", false},
		{"empty content", "test-empty.txt", "", false},
		{"unicode content", "test-unicode.txt", "Hello 世界 🌍", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up after test
			defer func() {
				if err := os.RemoveAll(filepath.Dir(tt.path)); err != nil {
					t.Logf("Warning: failed to remove test directory: %v", err)
				}
			}()

			err := WriteFile(tt.path, tt.content)
			if tt.expectError {
				if err == nil {
					t.Errorf("WriteFile(%s, %s) expected error but got nil", tt.path, tt.content)
				}
			} else {
				if err != nil {
					t.Errorf("WriteFile(%s, %s) expected no error but got: %v", tt.path, tt.content, err)
				}
				// Verify file was created and content is correct
				if !FileExists(tt.path) {
					t.Errorf("WriteFile(%s) should have created file", tt.path)
				}
				readContent, err := ReadFile(tt.path)
				if err != nil {
					t.Errorf("WriteFile(%s) created file but ReadFile failed: %v", tt.path, err)
				}
				if readContent != tt.content {
					t.Errorf("WriteFile(%s) content mismatch. Got: %q, Expected: %q", tt.path, readContent, tt.content)
				}
			}
		})
	}
}

func TestReadFile(t *testing.T) {
	// Create a temporary file with known content
	content := "Test content with\nmultiple lines\nand unicode: 世界 🌍"
	tmpFile, err := os.CreateTemp("", "test-read-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Warning: failed to remove temp file: %v", err)
		}
	}()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		expected    string
		expectError bool
	}{
		{"existing file", tmpFile.Name(), content, false},
		{"non-existing file", "non-existing-file.txt", "", true},
		{"non-existing path", "non/existing/path.txt", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ReadFile(tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("ReadFile(%s) expected error but got nil", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("ReadFile(%s) expected no error but got: %v", tt.path, err)
				}
				if result != tt.expected {
					t.Errorf("ReadFile(%s) = %q, expected %q", tt.path, result, tt.expected)
				}
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	// Create source file
	sourceContent := "Source file content\nwith multiple lines\nand unicode: 世界 🌍"
	sourceFile, err := os.CreateTemp("", "test-copy-source-*.txt")
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	defer func() {
		if err := os.Remove(sourceFile.Name()); err != nil {
			t.Logf("Warning: failed to remove source file: %v", err)
		}
	}()

	if _, err := sourceFile.WriteString(sourceContent); err != nil {
		t.Fatalf("Failed to write to source file: %v", err)
	}
	if err := sourceFile.Close(); err != nil {
		t.Fatalf("Failed to close source file: %v", err)
	}

	tests := []struct {
		name        string
		src         string
		dst         string
		expectError bool
	}{
		{"valid copy", sourceFile.Name(), "test-copy-dest.txt", false},
		{"copy to nested directory", sourceFile.Name(), "nested/test-copy-dest.txt", false},
		{"non-existing source", "non-existing-source.txt", "test-copy-dest.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up after test
			defer func() {
				if err := os.RemoveAll(filepath.Dir(tt.dst)); err != nil {
					t.Logf("Warning: failed to remove test directory: %v", err)
				}
			}()

			err := CopyFile(tt.src, tt.dst)
			if tt.expectError {
				if err == nil {
					t.Errorf("CopyFile(%s, %s) expected error but got nil", tt.src, tt.dst)
				}
			} else {
				if err != nil {
					t.Errorf("CopyFile(%s, %s) expected no error but got: %v", tt.src, tt.dst, err)
				}
				// Verify destination file was created and content matches
				if !FileExists(tt.dst) {
					t.Errorf("CopyFile(%s, %s) should have created destination file", tt.src, tt.dst)
				}
				destContent, err := ReadFile(tt.dst)
				if err != nil {
					t.Errorf("CopyFile(%s, %s) created file but ReadFile failed: %v", tt.src, tt.dst, err)
				}
				if destContent != sourceContent {
					t.Errorf("CopyFile(%s, %s) content mismatch. Got: %q, Expected: %q", tt.src, tt.dst, destContent, sourceContent)
				}
			}
		})
	}
}

func TestListFiles(t *testing.T) {
	// Create test directory structure
	testDir, err := os.MkdirTemp("", "test-list-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Logf("Warning: failed to remove test directory: %v", err)
		}
	}()

	// Create test files
	files := []string{
		"file1.txt",
		"file2.txt",
		"file3.md",
		"subdir/file4.txt",
		"subdir/file5.md",
	}

	for _, file := range files {
		filePath := filepath.Join(testDir, file)
		if err := CreateDirectory(filepath.Dir(filePath)); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", file, err)
		}
		if err := WriteFile(filePath, "test content"); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	tests := []struct {
		name     string
		dir      string
		ext      string
		expected int // Expected number of files
	}{
		{"txt files", testDir, ".txt", 3},
		{"md files", testDir, ".md", 2},
		{"go files", testDir, ".go", 0},
		{"all files", testDir, "", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := ListFiles(tt.dir, tt.ext)
			if err != nil {
				t.Errorf("ListFiles(%s, %s) expected no error but got: %v", tt.dir, tt.ext, err)
			}
			if len(files) != tt.expected {
				t.Errorf("ListFiles(%s, %s) = %d files, expected %d", tt.dir, tt.ext, len(files), tt.expected)
			}
		})
	}
}

func TestGetRelativePath(t *testing.T) {
	tests := []struct {
		name        string
		base        string
		target      string
		expected    string
		expectError bool
	}{
		{"simple relative", "/home/user", "/home/user/project", "project", false},
		{"nested relative", "/home/user", "/home/user/project/subdir", "project/subdir", false},
		{"same directory", "/home/user/project", "/home/user/project", ".", false},
		{"parent directory", "/home/user/project", "/home/user", "..", false},
		{"different roots", "/home/user", "/var/log", "../../var/log", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetRelativePath(tt.base, tt.target)
			if tt.expectError {
				if err == nil {
					t.Errorf("GetRelativePath(%s, %s) expected error but got nil", tt.base, tt.target)
				}
			} else {
				if err != nil {
					t.Errorf("GetRelativePath(%s, %s) expected no error but got: %v", tt.base, tt.target, err)
				}
				if result != tt.expected {
					t.Errorf("GetRelativePath(%s, %s) = %q, expected %q", tt.base, tt.target, result, tt.expected)
				}
			}
		})
	}
}

func TestDeleteFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-delete-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	t.Run("delete existing file", func(t *testing.T) {
		err := DeleteFile(tmpPath)
		if err != nil {
			t.Errorf("DeleteFile(%s) expected no error but got: %v", tmpPath, err)
		}
		if FileExists(tmpPath) {
			t.Errorf("DeleteFile(%s) should have removed file", tmpPath)
		}
	})

	t.Run("delete non-existing file", func(t *testing.T) {
		err := DeleteFile("non-existing-file-delete-test.txt")
		if err == nil {
			t.Error("DeleteFile(non-existing) expected error but got nil")
		}
	})
}

func TestCleanPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"a/../b", "b"},
		{".", "."},
		{"", "."},
		{"a/b/c", "a/b/c"},
		{"a//b", "a/b"},
		{"a/./b", "a/b"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := CleanPath(tt.path)
			if got != tt.expected {
				t.Errorf("CleanPath(%q) = %q, expected %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name     string
		elem     []string
		expected string
	}{
		{"single", []string{"a"}, "a"},
		{"two", []string{"a", "b"}, "a/b"},
		{"three", []string{"a", "b", "c"}, "a/b/c"},
		{"empty segments", []string{"a", "", "c"}, "a/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JoinPath(tt.elem...)
			if got != tt.expected {
				t.Errorf("JoinPath(%v) = %q, expected %q", tt.elem, got, tt.expected)
			}
		})
	}
}

func TestIsGoCloudGenerated(t *testing.T) {
	tmpDir := t.TempDir()

	withSignature := filepath.Join(tmpDir, "with-sig.txt")
	if err := os.WriteFile(withSignature, []byte("header\nThis file is generated and maintained by GoCloud CLI\nfooter"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	withoutSignature := filepath.Join(tmpDir, "without-sig.txt")
	if err := os.WriteFile(withoutSignature, []byte("just some content"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"file with signature", withSignature, true},
		{"file without signature", withoutSignature, false},
		{"non-existing file", filepath.Join(tmpDir, "nonexistent.txt"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGoCloudGenerated(tt.path)
			if got != tt.expected {
				t.Errorf("IsGoCloudGenerated(%s) = %v, expected %v", tt.path, got, tt.expected)
			}
		})
	}
}
