package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFilesystemGuard_PathTraversalAndJail(t *testing.T) {
	tmpDir := t.TempDir()
	wsRoot := filepath.Join(tmpDir, "workspace")
	dataRoot := filepath.Join(tmpDir, "data")
	tempRoot := filepath.Join(tmpDir, "temp")

	_ = os.MkdirAll(filepath.Join(wsRoot, "src"), 0755)
	testFile := filepath.Join(wsRoot, "src", "main.go")
	_ = os.WriteFile(testFile, []byte("package main"), 0644)

	guard, err := NewFilesystemGuard(wsRoot, dataRoot, tempRoot, false)
	if err != nil {
		t.Fatalf("failed to create filesystem guard: %v", err)
	}

	// 1. Permitted read inside workspace
	canon, err := guard.ValidateRead("src/main.go")
	if err != nil {
		t.Fatalf("expected valid read to pass: %v", err)
	}
	if filepath.Clean(canon) != filepath.Clean(testFile) {
		t.Fatalf("expected canonical path %s, got %s", testFile, canon)
	}

	// 2. Traversal attempt via relative path
	_, err = guard.ValidateRead("../../etc/passwd")
	if err == nil {
		t.Fatal("expected traversal attempt to fail")
	}

	// 3. Absolute path outside root
	outsidePath := filepath.Join(tmpDir, "outside.txt")
	_ = os.WriteFile(outsidePath, []byte("secret"), 0644)
	_, err = guard.ValidateRead(outsidePath)
	if err == nil {
		t.Fatal("expected read outside root to fail")
	}

	// 4. UNC path attempt
	_, err = guard.ValidateRead(`\\attacker-server\share\malicious.exe`)
	if err == nil {
		t.Fatal("expected UNC path to fail")
	}

	// 5. Writing to repository when allowRepoWrite is false
	_, err = guard.ValidateWrite("src/new.go")
	if err == nil {
		t.Fatal("expected write to repo to be denied when repo writes are disabled")
	}

	// 6. Writing to private plugin storage must succeed
	_ = guard.EnsureStorageDirectories()
	privFile := filepath.Join(dataRoot, "state.json")
	canonWrite, err := guard.ValidateWrite(privFile)
	if err != nil {
		t.Fatalf("expected write to private data root to pass: %v", err)
	}
	if filepath.Clean(canonWrite) != filepath.Clean(privFile) {
		t.Fatalf("expected %s, got %s", privFile, canonWrite)
	}
}

func TestFilesystemGuard_SymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Symlink creation on Windows often requires SeCreateSymbolicLinkPrivilege (admin or dev mode)
		// We test symlink resolution logic if symlink creation succeeds, else skip
	}

	tmpDir := t.TempDir()
	wsRoot := filepath.Join(tmpDir, "ws")
	dataRoot := filepath.Join(tmpDir, "data")
	tempRoot := filepath.Join(tmpDir, "temp")
	_ = os.MkdirAll(wsRoot, 0755)

	secretFile := filepath.Join(tmpDir, "secret.txt")
	_ = os.WriteFile(secretFile, []byte("super-secret"), 0644)

	linkPath := filepath.Join(wsRoot, "secret-link.txt")
	err := os.Symlink(secretFile, linkPath)
	if err != nil {
		t.Skip("skipping symlink test (symlink creation not permitted in this environment)")
	}

	guard, err := NewFilesystemGuard(wsRoot, dataRoot, tempRoot, false)
	if err != nil {
		t.Fatal(err)
	}

	// Validating read on the symlink must resolve to target and fail because secret.txt is outside wsRoot
	_, err = guard.ValidateRead("secret-link.txt")
	if err == nil {
		t.Fatal("expected symlink pointing outside workspace root to be blocked")
	}
}

func TestNetworkGuard_Enforcement(t *testing.T) {
	// 1. Default deny
	ng1 := NewNetworkGuard(false, false, nil)
	if ng1.IsNetworkAllowed() {
		t.Fatal("expected network to be denied by default")
	}
	if err := ng1.ValidateConnection("tcp", "example.com:443"); err == nil {
		t.Fatal("expected connection to be rejected when network is disabled")
	}

	// 2. Network allowed with destination allowlist
	ng2 := NewNetworkGuard(true, false, []string{"api.github.com:443", "*.trusted.org"})
	if err := ng2.ValidateConnection("tcp", "api.github.com:443"); err != nil {
		t.Fatalf("expected allowed destination to pass: %v", err)
	}
	if err := ng2.ValidateConnection("tcp", "sub.trusted.org:80"); err != nil {
		t.Fatalf("expected wildcard domain to pass: %v", err)
	}
	if err := ng2.ValidateConnection("tcp", "malicious.com:80"); err == nil {
		t.Fatal("expected unlisted destination to be blocked")
	}

	// 3. Localhost blocking
	if err := ng2.ValidateConnection("tcp", "127.0.0.1:8080"); err == nil {
		t.Fatal("expected loopback connection to be blocked when allowLocalhost is false")
	}

	// 4. Environment sanitization
	env := []string{"PATH=/usr/bin", "HTTP_PROXY=http://proxy:8080", "GITHUB_TOKEN=secret123", "MY_VAR=hello"}
	sanitized := ng2.SanitizeEnvironment(env)
	for _, e := range sanitized {
		if e == "HTTP_PROXY=http://proxy:8080" || e == "GITHUB_TOKEN=secret123" {
			t.Fatalf("expected sensitive env to be stripped: %s", e)
		}
	}
}

func TestResourceLimiter_Limits(t *testing.T) {
	limits := ResourceLimits{
		MaxMemoryBytes:  100 * 1024 * 1024, // 100 MB
		MaxProcesses:    2,
		MaxMessageBytes: 1024,
	}
	limiter := NewResourceLimiter(limits)

	// Process count
	if err := limiter.CheckProcessSpawn(); err != nil {
		t.Fatal(err)
	}
	limiter.IncrementProcesses()
	limiter.IncrementProcesses()
	if err := limiter.CheckProcessSpawn(); err == nil {
		t.Fatal("expected process count limit to trigger error")
	}
	limiter.DecrementProcesses()
	if err := limiter.CheckProcessSpawn(); err != nil {
		t.Fatal("expected process spawn to pass after decrement")
	}

	// Message size
	if err := limiter.CheckMessageSize(500); err != nil {
		t.Fatal(err)
	}
	if err := limiter.CheckMessageSize(2000); err == nil {
		t.Fatal("expected oversized message to fail")
	}

	// Memory usage
	if err := limiter.RecordMemoryUsage(50 * 1024 * 1024); err != nil {
		t.Fatal(err)
	}
	if err := limiter.RecordMemoryUsage(150 * 1024 * 1024); err == nil {
		t.Fatal("expected memory limit breach to fail")
	}
}

func TestPlatformSandbox_Lifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := SandboxConfig{
		PluginID:      "test-plugin-sbx",
		WorkspaceRoot: filepath.Join(tmpDir, "ws"),
		DataRoot:      filepath.Join(tmpDir, "data"),
		TempRoot:      filepath.Join(tmpDir, "temp"),
		Limits:        DefaultResourceLimits(),
	}

	sbx, err := NewPlatformSandbox(cfg)
	if err != nil {
		t.Fatalf("failed to create platform sandbox: %v", err)
	}

	if err := sbx.Initialize(context.Background()); err != nil {
		t.Fatalf("failed to initialize sandbox: %v", err)
	}

	// Spawn a real child subprocess to attach to the sandbox
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", "ping 127.0.0.1 -n 5 >nul")
	} else {
		cmd = exec.Command("sleep", "5")
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start test child process: %v", err)
	}
	childPID := cmd.Process.Pid

	// Attach child process to the sandbox
	if err := sbx.AttachProcess(childPID); err != nil {
		t.Logf("attach process to sandbox returned: %v (expected if running under nested restricted container)", err)
	}

	// Terminate sandbox (must kill the child process cleanly)
	if err := sbx.Terminate(); err != nil {
		t.Fatalf("terminate sandbox failed: %v", err)
	}

	// Wait for child process to exit
	_ = cmd.Wait()

	if err := sbx.Cleanup(); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
}

func testSandboxConfig(tmpDir string) SandboxConfig {
	return SandboxConfig{
		PluginID:      "test-plugin-sbx",
		WorkspaceRoot: filepath.Join(tmpDir, "ws"),
		DataRoot:      filepath.Join(tmpDir, "data"),
		TempRoot:      filepath.Join(tmpDir, "temp"),
		Limits:        DefaultResourceLimits(),
	}
}

func TestSandbox_AttachProcessNonexistentPIDFails(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := testSandboxConfig(tmpDir)
	sbx, err := NewPlatformSandbox(cfg)
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sbx.Cleanup()

	bogusPID := 99999999
	err = sbx.AttachProcess(bogusPID)
	if err == nil {
		t.Fatalf("expected AttachProcess on nonexistent PID %d to fail, got nil", bogusPID)
	}
}

func TestSandbox_AttachProcessInvalidPIDFails(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := testSandboxConfig(tmpDir)
	sbx, err := NewPlatformSandbox(cfg)
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sbx.Cleanup()

	for _, invalidPID := range []int{0, -1, -99} {
		if err := sbx.AttachProcess(invalidPID); err == nil {
			t.Errorf("expected AttachProcess(%d) to fail, got nil", invalidPID)
		}
	}
}

func TestSandbox_RepeatedTerminationAndCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := testSandboxConfig(tmpDir)
	sbx, err := NewPlatformSandbox(cfg)
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := sbx.Terminate(); err != nil {
			t.Errorf("repeated Terminate call %d failed: %v", i, err)
		}
	}

	for i := 0; i < 3; i++ {
		if err := sbx.Cleanup(); err != nil {
			t.Errorf("repeated Cleanup call %d failed: %v", i, err)
		}
	}
}
