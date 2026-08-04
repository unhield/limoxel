package filesystem_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/filesystem"
)

func TestOSFilesystemOperations(t *testing.T) {
	tempDir := t.TempDir()
	fs := filesystem.NewOSFilesystem()

	filePath := filepath.Join(tempDir, "test.txt")
	subDir := filepath.Join(tempDir, "subdir")

	// 1. Exists
	t.Run("Exists", func(t *testing.T) {
		if fs.Exists(filePath) {
			t.Error("file should not exist yet")
		}
		if !fs.Exists(tempDir) {
			t.Error("tempDir should exist")
		}
		var nilFs *filesystem.OSFilesystem
		if nilFs.Exists(tempDir) {
			t.Error("expected false for nil filesystem Exists")
		}
	})

	// 2. WriteFile & ReadFile
	t.Run("WriteFile and ReadFile", func(t *testing.T) {
		content := []byte("hello filesystem")
		if err := fs.WriteFile(filePath, content, 0644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}

		got, err := fs.ReadFile(filePath)
		if err != nil || string(got) != "hello filesystem" {
			t.Errorf("ReadFile got %q, %v; want 'hello filesystem', nil", string(got), err)
		}

		// Read non-existent
		_, err = fs.ReadFile(filepath.Join(tempDir, "missing.txt"))
		if !errors.Is(err, filesystem.ErrFileNotFound) {
			t.Errorf("got %v, want ErrFileNotFound", err)
		}
	})

	// 3. Stat & Entry getters
	t.Run("Stat and Entry", func(t *testing.T) {
		entry, err := fs.Stat(filePath)
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}
		if entry.Path() != filePath {
			t.Errorf("got path %q, want %q", entry.Path(), filePath)
		}
		if entry.IsDir() {
			t.Error("file should not be directory")
		}
		if entry.Size() != int64(len("hello filesystem")) {
			t.Errorf("got size %d, want %d", entry.Size(), len("hello filesystem"))
		}
		if entry.ModTime().IsZero() {
			t.Error("expected non-zero ModTime")
		}

		dirEntry, err := fs.Stat(tempDir)
		if err != nil || !dirEntry.IsDir() {
			t.Errorf("dir Stat failed or is not dir: %v", err)
		}

		var nilEntry *filesystem.Entry
		if nilEntry.Path() != "" || nilEntry.IsDir() || nilEntry.Size() != 0 || !nilEntry.ModTime().IsZero() {
			t.Error("expected zero values for nil Entry getters")
		}
	})

	// 4. MkdirAll & ReadDir
	t.Run("MkdirAll and ReadDir", func(t *testing.T) {
		if err := fs.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}

		entries, err := fs.ReadDir(tempDir)
		if err != nil {
			t.Fatalf("ReadDir failed: %v", err)
		}

		if len(entries) != 2 {
			t.Fatalf("got %d entries, want 2", len(entries))
		}
	})

	// 5. Rename
	t.Run("Rename", func(t *testing.T) {
		renamedPath := filepath.Join(tempDir, "renamed.txt")
		if err := fs.Rename(filePath, renamedPath); err != nil {
			t.Fatalf("Rename failed: %v", err)
		}
		if fs.Exists(filePath) {
			t.Error("old file path should not exist after rename")
		}
		if !fs.Exists(renamedPath) {
			t.Error("new file path should exist after rename")
		}
	})

	// 6. Remove & RemoveAll
	t.Run("Remove and RemoveAll", func(t *testing.T) {
		renamedPath := filepath.Join(tempDir, "renamed.txt")
		if err := fs.Remove(renamedPath); err != nil {
			t.Fatalf("Remove failed: %v", err)
		}
		if fs.Exists(renamedPath) {
			t.Error("renamed file should be removed")
		}

		if err := fs.RemoveAll(subDir); err != nil {
			t.Fatalf("RemoveAll failed: %v", err)
		}
		if fs.Exists(subDir) {
			t.Error("subDir should be removed")
		}
	})

	// 7. Nil Filesystem Safety
	t.Run("Nil OSFilesystem Safety", func(t *testing.T) {
		var nilFs *filesystem.OSFilesystem
		if _, err := nilFs.Stat(filePath); !errors.Is(err, filesystem.ErrNilFilesystem) {
			t.Errorf("got %v, want ErrNilFilesystem", err)
		}
		if _, err := nilFs.ReadFile(filePath); !errors.Is(err, filesystem.ErrNilFilesystem) {
			t.Errorf("got %v, want ErrNilFilesystem", err)
		}
		if err := nilFs.WriteFile(filePath, nil, 0644); !errors.Is(err, filesystem.ErrNilFilesystem) {
			t.Errorf("got %v, want ErrNilFilesystem", err)
		}
		if _, err := nilFs.ReadDir(tempDir); !errors.Is(err, filesystem.ErrNilFilesystem) {
			t.Errorf("got %v, want ErrNilFilesystem", err)
		}
		if err := nilFs.MkdirAll(tempDir, 0755); !errors.Is(err, filesystem.ErrNilFilesystem) {
			t.Errorf("got %v, want ErrNilFilesystem", err)
		}
		if err := nilFs.Remove(filePath); !errors.Is(err, filesystem.ErrNilFilesystem) {
			t.Errorf("got %v, want ErrNilFilesystem", err)
		}
		if err := nilFs.RemoveAll(tempDir); !errors.Is(err, filesystem.ErrNilFilesystem) {
			t.Errorf("got %v, want ErrNilFilesystem", err)
		}
		if err := nilFs.Rename(filePath, tempDir); !errors.Is(err, filesystem.ErrNilFilesystem) {
			t.Errorf("got %v, want ErrNilFilesystem", err)
		}
	})
}

func TestIgnorer(t *testing.T) {
	ign := filesystem.NewIgnorer("custom_ignore", ".tmp")

	if !ign.ShouldIgnore("/project/.git/HEAD") {
		t.Error("expected .git segment to be ignored")
	}
	if !ign.ShouldIgnore("/project/node_modules/package.json") {
		t.Error("expected node_modules to be ignored")
	}
	if !ign.ShouldIgnore("/project/build/custom_ignore") {
		t.Error("expected custom_ignore to be ignored")
	}
	if !ign.ShouldIgnore("Thumbs.db") {
		t.Error("expected Thumbs.db to be ignored")
	}
	if ign.ShouldIgnore("/project/src/main.go") {
		t.Error("src/main.go should not be ignored")
	}

	rules := ign.Rules()
	if len(rules) < 12 {
		t.Errorf("expected at least 12 rules, got %d", len(rules))
	}
	// Verify defensive copy
	rules[0] = "mutated"
	if ign.Rules()[0] == "mutated" {
		t.Error("rules list mutation leaked to ignorer")
	}

	var nilIgn *filesystem.Ignorer
	if nilIgn.ShouldIgnore("/project/.git") {
		t.Error("nil ignorer ShouldIgnore should return false")
	}
	if nilIgn.Rules() != nil {
		t.Error("nil ignorer Rules should return nil")
	}
}

func TestDiscoverer(t *testing.T) {
	tempDir := t.TempDir()

	// Create directory structure:
	// tempDir/
	//   a.txt
	//   b/
	//     c.txt
	//   .git/
	//     config
	_ = os.WriteFile(filepath.Join(tempDir, "a.txt"), []byte("a"), 0644)
	bDir := filepath.Join(tempDir, "b")
	_ = os.MkdirAll(bDir, 0755)
	_ = os.WriteFile(filepath.Join(bDir, "c.txt"), []byte("c"), 0644)

	gitDir := filepath.Join(tempDir, ".git")
	_ = os.MkdirAll(gitDir, 0755)
	_ = os.WriteFile(filepath.Join(gitDir, "config"), []byte("git"), 0644)

	ign := filesystem.NewIgnorer()
	disc, err := filesystem.NewDiscoverer(tempDir, ign)
	if err != nil {
		t.Fatalf("NewDiscoverer failed: %v", err)
	}

	if disc.Root() == "" {
		t.Error("expected non-empty Root")
	}
	if disc.Ignorer() != ign {
		t.Error("expected ignorer match")
	}

	res, err := disc.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if res.Root() != disc.Root() {
		t.Errorf("got root %q, want %q", res.Root(), disc.Root())
	}

	// Discovered entries should include tempDir, a.txt, b, b/c.txt but exclude .git and .git/config
	entries := res.Entries()
	if res.Count() != len(entries) {
		t.Errorf("Count mismatch: count=%d, len(entries)=%d", res.Count(), len(entries))
	}

	for _, entry := range entries {
		if filesystem.NewIgnorer().ShouldIgnore(entry.Path()) {
			t.Errorf("discovered path %q should have been ignored", entry.Path())
		}
	}

	// Verify deterministic sorting (entries sorted by path)
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Path() >= entries[i].Path() {
			t.Errorf("entries out of deterministic order at index %d: %s >= %s", i, entries[i-1].Path(), entries[i].Path())
		}
	}

	// Test NewDiscoverer validation errors
	_, err = filesystem.NewDiscoverer("")
	if !errors.Is(err, filesystem.ErrPathEmpty) {
		t.Errorf("got %v, want ErrPathEmpty", err)
	}

	_, err = filesystem.NewDiscoverer(filepath.Join(tempDir, "missing"))
	if !errors.Is(err, filesystem.ErrRootNotFound) {
		t.Errorf("got %v, want ErrRootNotFound", err)
	}

	filePath := filepath.Join(tempDir, "a.txt")
	_, err = filesystem.NewDiscoverer(filePath)
	if !errors.Is(err, filesystem.ErrNotDirectory) {
		t.Errorf("got %v, want ErrNotDirectory", err)
	}

	var nilDisc *filesystem.Discoverer
	if _, err := nilDisc.Discover(); !errors.Is(err, filesystem.ErrNilDiscoverer) {
		t.Errorf("got %v, want ErrNilDiscoverer", err)
	}
	if nilDisc.Root() != "" || nilDisc.Ignorer() != nil {
		t.Error("expected zero values for nil Discoverer getters")
	}

	var nilRes *filesystem.Result
	if nilRes.Root() != "" || nilRes.Count() != 0 || nilRes.Entries() != nil {
		t.Error("expected zero values for nil Result getters")
	}
}

func TestFileService(t *testing.T) {
	tempDir := t.TempDir()
	osFs := filesystem.NewOSFilesystem()
	svc, err := filesystem.NewFileService(osFs)
	if err != nil {
		t.Fatalf("NewFileService failed: %v", err)
	}

	// 1. WriteFile & ReadFile
	file1 := filepath.Join(tempDir, "svc1.txt")
	if err := svc.WriteFile(file1, []byte("data"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if !svc.Exists(file1) {
		t.Error("svc1.txt should exist")
	}
	data, err := svc.ReadFile(file1)
	if err != nil || string(data) != "data" {
		t.Errorf("ReadFile got %q, %v", string(data), err)
	}

	// 2. EnsureDirectory
	subDir := filepath.Join(tempDir, "nested", "dir")
	if err := svc.EnsureDirectory(subDir); err != nil {
		t.Fatalf("EnsureDirectory failed: %v", err)
	}
	if !svc.Exists(subDir) {
		t.Error("subDir should exist after EnsureDirectory")
	}
	// Calling EnsureDirectory on existing directory should succeed
	if err := svc.EnsureDirectory(subDir); err != nil {
		t.Errorf("EnsureDirectory on existing dir failed: %v", err)
	}
	// Calling EnsureDirectory on file path should fail
	if err := svc.EnsureDirectory(file1); !errors.Is(err, filesystem.ErrNotDirectory) {
		t.Errorf("got %v, want ErrNotDirectory", err)
	}

	// 3. CopyFile
	copyDest := filepath.Join(tempDir, "copy_target", "svc1_copy.txt")
	if err := svc.CopyFile(file1, copyDest); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}
	copyData, err := svc.ReadFile(copyDest)
	if err != nil || string(copyData) != "data" {
		t.Errorf("CopyFile content got %q, %v", string(copyData), err)
	}

	// Copy missing source
	if err := svc.CopyFile(filepath.Join(tempDir, "missing.txt"), copyDest); !errors.Is(err, filesystem.ErrSourceNotFound) {
		t.Errorf("got %v, want ErrSourceNotFound", err)
	}

	// 4. Rename & Remove
	file2 := filepath.Join(tempDir, "svc2.txt")
	if err := svc.Rename(file1, file2); err != nil {
		t.Fatalf("Rename failed: %v", err)
	}
	if svc.Exists(file1) {
		t.Error("file1 should be renamed")
	}
	if err := svc.Remove(file2); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if err := svc.RemoveAll(tempDir); err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}

	// 5. Nil Service & Nil Filesystem Safety
	_, err = filesystem.NewFileService(nil)
	if !errors.Is(err, filesystem.ErrNilFilesystem) {
		t.Errorf("got %v, want ErrNilFilesystem", err)
	}

	var nilSvc *filesystem.FileService
	if nilSvc.Exists(file1) {
		t.Error("expected false for nil service Exists")
	}
	if _, err := nilSvc.ReadFile(file1); !errors.Is(err, filesystem.ErrNilService) {
		t.Errorf("got %v, want ErrNilService", err)
	}
	if err := nilSvc.WriteFile(file1, nil, 0644); !errors.Is(err, filesystem.ErrNilService) {
		t.Errorf("got %v, want ErrNilService", err)
	}
	if err := nilSvc.MkdirAll(tempDir, 0755); !errors.Is(err, filesystem.ErrNilService) {
		t.Errorf("got %v, want ErrNilService", err)
	}
	if err := nilSvc.EnsureDirectory(tempDir); !errors.Is(err, filesystem.ErrNilService) {
		t.Errorf("got %v, want ErrNilService", err)
	}
	if err := nilSvc.Remove(file1); !errors.Is(err, filesystem.ErrNilService) {
		t.Errorf("got %v, want ErrNilService", err)
	}
	if err := nilSvc.RemoveAll(tempDir); !errors.Is(err, filesystem.ErrNilService) {
		t.Errorf("got %v, want ErrNilService", err)
	}
	if err := nilSvc.Rename(file1, file2); !errors.Is(err, filesystem.ErrNilService) {
		t.Errorf("got %v, want ErrNilService", err)
	}
	if err := nilSvc.CopyFile(file1, file2); !errors.Is(err, filesystem.ErrNilService) {
		t.Errorf("got %v, want ErrNilService", err)
	}
}

func TestNewEntryConstructor(t *testing.T) {
	now := time.Now()
	entry := filesystem.NewEntry("/a/b/c", true, 1024, now)
	if entry.Path() != filepath.Clean("/a/b/c") {
		t.Errorf("got path %q", entry.Path())
	}
	if !entry.IsDir() {
		t.Error("expected IsDir true")
	}
	if entry.Size() != 1024 {
		t.Errorf("got size %d, want 1024", entry.Size())
	}
	if entry.ModTime() != now {
		t.Errorf("got ModTime %v, want %v", entry.ModTime(), now)
	}
}
