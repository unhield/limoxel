package sandbox

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
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

	// Isolate process before launch so it runs in its own process group on Unix
	IsolateProcessCmd(cmd)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start test child process: %v", err)
	}
	childPID := cmd.Process.Pid

	// Attach child process to the sandbox
	if err := sbx.AttachProcess(childPID); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("attach process to sandbox failed: %v", err)
	}

	// Terminate sandbox (must kill the child process cleanly without killing test runner)
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

func TestResourceLimiter_ConcurrentIncrementDecrement(t *testing.T) {
	limits := ResourceLimits{
		MaxProcesses: 50,
	}
	limiter := NewResourceLimiter(limits)

	var wg sync.WaitGroup
	workers := 100
	iterations := 200

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				limiter.IncrementProcesses()
				limiter.DecrementProcesses()
			}
		}()
	}
	wg.Wait()

	if active := limiter.ActiveProcesses(); active != 0 {
		t.Fatalf("expected 0 active processes after balanced inc/dec, got %d", active)
	}

	// Ensure calling DecrementProcesses at 0 does not underflow uint32
	limiter.DecrementProcesses()
	if active := limiter.ActiveProcesses(); active != 0 {
		t.Fatalf("expected 0 active processes after decrement at zero, got %d (underflow!)", active)
	}
}

func TestResourceLimiter_MaxProcessesCAS(t *testing.T) {
	limits := ResourceLimits{
		MaxProcesses: 1,
	}
	limiter := NewResourceLimiter(limits)

	// Single slot: exactly 1 caller should successfully reserve among 50 goroutines
	var wg sync.WaitGroup
	var successfulReservations int64
	workers := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := limiter.ReserveProcess(); err == nil {
				atomic.AddInt64(&successfulReservations, 1)
			}
		}()
	}
	wg.Wait()

	if successfulReservations != 1 {
		t.Fatalf("expected exactly 1 successful reservation, got %d", successfulReservations)
	}
	if limiter.ActiveProcesses() != 1 {
		t.Fatalf("expected 1 active process, got %d", limiter.ActiveProcesses())
	}

	// Release it
	limiter.ReleaseProcess()
	if limiter.ActiveProcesses() != 0 {
		t.Fatalf("expected 0 active processes after release, got %d", limiter.ActiveProcesses())
	}

	// Now another reservation should succeed
	if err := limiter.ReserveProcess(); err != nil {
		t.Fatalf("expected reservation to succeed after release, got %v", err)
	}
}

func TestFilesystemGuard_SensitiveMetadataCasingAndNested(t *testing.T) {
	tmpDir := t.TempDir()
	wsRoot := filepath.Join(tmpDir, "workspace")
	dataRoot := filepath.Join(tmpDir, "data")
	tempRoot := filepath.Join(tmpDir, "temp")

	_ = os.MkdirAll(filepath.Join(wsRoot, "sub", "dir"), 0755)

	// Even with allowRepoWrite = true, sensitive metadata paths must be protected
	guard, err := NewFilesystemGuard(wsRoot, dataRoot, tempRoot, true)
	if err != nil {
		t.Fatalf("failed to create guard: %v", err)
	}

	sensitiveTargets := []string{
		".git",
		filepath.Join(".git", "config"),
		".GIT",
		filepath.Join(".Git", "HEAD"),
		".limoxel",
		filepath.Join(".limoxel", "config.json"),
		".LiMoXeL",
		filepath.Join("sub", "dir", ".git"),
		filepath.Join("sub", "dir", ".GIT"),
		filepath.Join("sub", "dir", ".limoxel"),
	}

	for _, target := range sensitiveTargets {
		_, err := guard.ValidateWrite(target)
		if err == nil {
			t.Errorf("expected sensitive path write to %q to fail, got nil", target)
		}
	}
}

func TestNetworkGuard_WildcardLabelBoundaries(t *testing.T) {
	ng := NewNetworkGuard(true, false, []string{"*.example.com"})

	allowed := []string{
		"api.example.com:443",
		"sub.example.com:80",
		"nested.sub.example.com:443",
	}
	for _, host := range allowed {
		if err := ng.ValidateConnection("tcp", host); err != nil {
			t.Errorf("expected %s to be allowed, got: %v", host, err)
		}
	}

	denied := []string{
		"example.com:443",
		"example.com.evil.com:443",
		"notexample.com:443",
		"evil-example.com:443",
		"badexample.com:80",
		"attacker.com:443",
	}
	for _, host := range denied {
		if err := ng.ValidateConnection("tcp", host); err == nil {
			t.Errorf("expected %s to be blocked, but was allowed", host)
		}
	}
}

func TestNetworkGuard_IPv4MappedIPv6AndLoopback(t *testing.T) {
	ng := NewNetworkGuard(true, false, []string{"*"}) // wildcard allowed, but localhost blocked

	loopbackTargets := []string{
		"127.0.0.1:8080",
		"127.0.0.2:80",
		"localhost:3000",
		"[::1]:8080",
		"::1:8080",
		"[::ffff:127.0.0.1]:8080",
		"::ffff:127.0.0.1:8080",
	}

	for _, target := range loopbackTargets {
		if err := ng.ValidateConnection("tcp", target); err == nil {
			t.Errorf("expected loopback %s to be blocked, but was allowed", target)
		}
	}
}

func TestSandbox_SelfProcessAttachmentDefense(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := testSandboxConfig(tmpDir)
	sbx, err := NewPlatformSandbox(cfg)
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sbx.Cleanup()

	if err := sbx.Initialize(context.Background()); err != nil {
		t.Fatalf("failed to initialize sandbox: %v", err)
	}

	// Attaching the current process (test runner) MUST be rejected with ErrSandboxViolation
	selfPID := os.Getpid()
	err = sbx.AttachProcess(selfPID)
	if err == nil {
		t.Fatalf("expected AttachProcess on current test runner PID %d to fail, got nil", selfPID)
	}
	if !errors.Is(err, ErrSandboxViolation) {
		t.Fatalf("expected ErrSandboxViolation, got: %v", err)
	}
}

func TestPlatformSandbox_AttachProcessEnforcesMaxProcesses(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := testSandboxConfig(tmpDir)
	cfg.Limits.MaxProcesses = 1

	sbx, err := NewPlatformSandbox(cfg)
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sbx.Cleanup()

	if err := sbx.Initialize(context.Background()); err != nil {
		t.Fatalf("failed to initialize sandbox: %v", err)
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", "ping 127.0.0.1 -n 5 >nul")
	} else {
		cmd = exec.Command("sleep", "5")
	}
	IsolateProcessCmd(cmd)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start test child: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// 1. First process attachment succeeds (quota: 1)
	if err := sbx.AttachProcess(cmd.Process.Pid); err != nil {
		t.Fatalf("first process attachment should succeed: %v", err)
	}

	// 2. Second attachment attempt exceeds MaxProcesses and must return ErrResourceExhausted
	err = sbx.AttachProcess(99999998)
	if err == nil {
		t.Fatal("expected second process attachment to fail due to MaxProcesses quota")
	}
	if !errors.Is(err, ErrResourceExhausted) {
		t.Fatalf("expected ErrResourceExhausted, got: %v", err)
	}

	// 3. Terminate sandbox and verify process slot is released
	if err := sbx.Terminate(); err != nil {
		t.Fatalf("terminate failed: %v", err)
	}
}
