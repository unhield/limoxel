package metadata_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/metadata"
	"github.com/unhield/limoxel/internal/language"
	"github.com/unhield/limoxel/internal/project"
	"github.com/unhield/limoxel/internal/repository"
	"github.com/unhield/limoxel/internal/workspace"
)

func setupTestLanguageRegistry(t *testing.T) *language.Registry {
	t.Helper()
	reg := language.NewRegistry()

	goLang, _ := language.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	pyLang, _ := language.New("python", "Python", []string{".py"}, nil, []string{"py"})
	tsLang, _ := language.New("typescript", "TypeScript", []string{".ts", ".tsx"}, nil, []string{"ts"})

	_ = reg.Register(goLang)
	_ = reg.Register(pyLang)
	_ = reg.Register(tsLang)

	return reg
}

func setupTestDiscoverer(t *testing.T) *discovery.Discoverer {
	t.Helper()
	reg := setupTestLanguageRegistry(t)
	disc, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("failed creating discoverer: %v", err)
	}
	return disc
}

func TestCollector_New(t *testing.T) {
	t.Run("nil discoverer returns ErrNilDiscoverer", func(t *testing.T) {
		col, err := metadata.New(nil)
		if col != nil {
			t.Errorf("expected nil collector, got %v", col)
		}
		if !errors.Is(err, metadata.ErrNilDiscoverer) {
			t.Errorf("expected ErrNilDiscoverer, got %v", err)
		}
	})

	t.Run("valid discoverer returns operational collector", func(t *testing.T) {
		disc := setupTestDiscoverer(t)
		col, err := metadata.New(disc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if col.Discoverer() != disc {
			t.Errorf("discoverer mismatch")
		}
	})
}

func TestCollector_InputValidation(t *testing.T) {
	disc := setupTestDiscoverer(t)
	col, err := metadata.New(disc)
	if err != nil {
		t.Fatalf("failed creating collector: %v", err)
	}

	t.Run("nil collector receiver methods return safe errors", func(t *testing.T) {
		var nilCol *metadata.Collector
		p, err := nilCol.Collect(nil)
		if p != nil || !errors.Is(err, metadata.ErrNilCollector) {
			t.Errorf("expected ErrNilCollector, got p=%v, err=%v", p, err)
		}

		p, err = nilCol.CollectRepository(nil)
		if p != nil || !errors.Is(err, metadata.ErrNilCollector) {
			t.Errorf("expected ErrNilCollector, got p=%v, err=%v", p, err)
		}

		p, err = nilCol.CollectPath("")
		if p != nil || !errors.Is(err, metadata.ErrNilCollector) {
			t.Errorf("expected ErrNilCollector, got p=%v, err=%v", p, err)
		}
	})

	t.Run("nil discovery result returns ErrNilDiscoveryResult", func(t *testing.T) {
		p, err := col.Collect(nil)
		if p != nil || !errors.Is(err, metadata.ErrNilDiscoveryResult) {
			t.Errorf("expected ErrNilDiscoveryResult, got p=%v, err=%v", p, err)
		}
	})

	t.Run("nil repository returns ErrNilRepository", func(t *testing.T) {
		p, err := col.CollectRepository(nil)
		if p != nil || !errors.Is(err, metadata.ErrNilRepository) {
			t.Errorf("expected ErrNilRepository, got p=%v, err=%v", p, err)
		}
	})

	t.Run("empty path returns ErrPathEmpty", func(t *testing.T) {
		p, err := col.CollectPath("")
		if p != nil || !errors.Is(err, metadata.ErrPathEmpty) {
			t.Errorf("expected ErrPathEmpty, got p=%v, err=%v", p, err)
		}

		p, err = col.CollectPath("   ")
		if p != nil || !errors.Is(err, metadata.ErrPathEmpty) {
			t.Errorf("expected ErrPathEmpty, got p=%v, err=%v", p, err)
		}
	})

	t.Run("nonexistent path returns ErrPathNotFound", func(t *testing.T) {
		p, err := col.CollectPath(filepath.Join(t.TempDir(), "nonexistent"))
		if p != nil || !errors.Is(err, metadata.ErrPathNotFound) {
			t.Errorf("expected ErrPathNotFound, got p=%v, err=%v", p, err)
		}
	})

	t.Run("file path instead of directory returns ErrNotDirectory", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "file.txt")
		_ = os.WriteFile(tempFile, []byte("content"), 0644)

		p, err := col.CollectPath(tempFile)
		if p != nil || !errors.Is(err, metadata.ErrNotDirectory) {
			t.Errorf("expected ErrNotDirectory, got p=%v, err=%v", p, err)
		}
	})
}

func TestCollector_FullGitRepositoryProfile(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	col, _ := metadata.New(disc)

	repoRoot := filepath.Join(tempDir, "full_repo")
	gitDir := filepath.Join(repoRoot, ".git")
	_ = os.MkdirAll(filepath.Join(gitDir, "refs", "heads"), 0755)
	_ = os.MkdirAll(filepath.Join(gitDir, "refs", "tags"), 0755)
	_ = os.MkdirAll(filepath.Join(gitDir, "refs", "remotes", "origin"), 0755)
	_ = os.MkdirAll(filepath.Join(gitDir, "logs"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "src"), 0755)

	// Mock git config with remote origin URL
	configContent := `[core]
	repositoryformatversion = 0
	bare = false
[remote "origin"]
	url = https://github.com/unhield/limoxel.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	_ = os.WriteFile(filepath.Join(gitDir, "config"), []byte(configContent), 0644)

	// Mock git HEAD & branches
	_ = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644)
	_ = os.WriteFile(filepath.Join(gitDir, "refs", "heads", "main"), []byte("1111222233334444555566667777888899990000\n"), 0644)
	_ = os.WriteFile(filepath.Join(gitDir, "refs", "remotes", "origin", "HEAD"), []byte("ref: refs/remotes/origin/main\n"), 0644)

	// Mock git reflog in .git/logs/HEAD
	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC).Unix()
	t2 := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC).Unix()
	t3 := time.Date(2026, 2, 1, 15, 0, 0, 0, time.UTC).Unix()

	logContent := "0000000000000000000000000000000000000000 aaaaa11111111111111111111111111111111111 Alice Engineer <alice@example.com> " +
		time.Unix(t1, 0).Format("1111111111 +0000") + "\tcommit (initial): first commit\n" +
		"aaaaa11111111111111111111111111111111111 bbbbb22222222222222222222222222222222222 Bob Developer <bob@example.com> " +
		time.Unix(t2, 0).Format("1111111111 +0000") + "\tcommit: add feature\n" +
		"bbbbb22222222222222222222222222222222222 1111222233334444555566667777888899990000 Alice Engineer <alice@example.com> " +
		time.Unix(t3, 0).Format("1111111111 +0000") + "\tcommit: release v1.0.0\n"

	_ = os.WriteFile(filepath.Join(gitDir, "logs", "HEAD"), []byte(logContent), 0644)

	// Mock loose and packed tags
	_ = os.WriteFile(filepath.Join(gitDir, "refs", "tags", "v0.1.0-alpha"), []byte("aaaaa11111111111111111111111111111111111\n"), 0644)
	packedRefs := "# pack-refs with: peeled-tags\n" +
		"1111222233334444555566667777888899990000 refs/tags/v1.0.0\n" +
		"^1111222233334444555566667777888899990000\n"
	_ = os.WriteFile(filepath.Join(gitDir, "packed-refs"), []byte(packedRefs), 0644)

	// Repository files
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "src", "util.py"), []byte("print(1)"), 0644)

	profile, err := col.CollectPath(repoRoot)
	if err != nil {
		t.Fatalf("CollectPath failed: %v", err)
	}

	if profile == nil {
		t.Fatal("expected non-nil profile")
	}

	// 1. Identity & Owner
	if profile.Name() != "full_repo" {
		t.Errorf("got name %q, want full_repo", profile.Name())
	}
	if profile.Owner() != "unhield" {
		t.Errorf("got owner %q, want unhield", profile.Owner())
	}
	if !profile.IsGit() {
		t.Error("expected isGit=true")
	}

	// 2. Branch
	if profile.CurrentBranch() != "main" {
		t.Errorf("got current branch %q, want main", profile.CurrentBranch())
	}
	if profile.DefaultBranch() != "main" {
		t.Errorf("got default branch %q, want main", profile.DefaultBranch())
	}

	// 3. Latest Commit
	latest := profile.LatestCommit()
	if latest == nil {
		t.Fatal("expected non-nil LatestCommit")
	}
	if latest.SHA() != "1111222233334444555566667777888899990000" {
		t.Errorf("got latest SHA %q", latest.SHA())
	}
	if latest.Author() != "Alice Engineer" || latest.Email() != "alice@example.com" {
		t.Errorf("got author %s <%s>", latest.Author(), latest.Email())
	}

	// 4. Commit Statistics
	stats := profile.CommitStats()
	if stats == nil {
		t.Fatal("expected non-nil CommitStats")
	}
	if stats.TotalCommits() != 3 {
		t.Errorf("got total commits %d, want 3", stats.TotalCommits())
	}
	if stats.EarliestSHA() != "aaaaa11111111111111111111111111111111111" {
		t.Errorf("got earliest SHA %q", stats.EarliestSHA())
	}

	// 5. Contributors (Alice: 2 commits, Bob: 1 commit)
	contribs := profile.Contributors()
	if len(contribs) != 2 {
		t.Fatalf("expected 2 contributors, got %d", len(contribs))
	}
	if contribs[0].Name() != "Alice Engineer" || contribs[0].CommitCount() != 2 {
		t.Errorf("expected top contributor Alice with 2 commits, got %+v", contribs[0])
	}
	if contribs[1].Name() != "Bob Developer" || contribs[1].CommitCount() != 1 {
		t.Errorf("expected second contributor Bob with 1 commit, got %+v", contribs[1])
	}

	// 6. Tags (v0.1.0-alpha, v1.0.0)
	tags := profile.Tags()
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	if tags[0].Name() != "v0.1.0-alpha" || tags[1].Name() != "v1.0.0" {
		t.Errorf("tags ordering/content mismatch: %+v, %+v", tags[0], tags[1])
	}

	// 7. Releases
	releases := profile.Releases()
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}

	// 8. Files and Languages reused from discovery
	if profile.TotalFiles() != 2 {
		t.Errorf("got total files %d, want 2", profile.TotalFiles())
	}
	if len(profile.Languages()) != 2 {
		t.Errorf("got languages count %d, want 2", len(profile.Languages()))
	}
}

func TestCollector_NonGitRepositoryProfile(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	col, _ := metadata.New(disc)

	repoRoot := filepath.Join(tempDir, "plain_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "app.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "script.py"), []byte("print('hi')"), 0644)

	profile, err := col.CollectPath(repoRoot)
	if err != nil {
		t.Fatalf("CollectPath failed: %v", err)
	}

	if profile.IsGit() {
		t.Error("expected isGit=false for non-git repository")
	}
	if profile.Owner() != "" {
		t.Errorf("expected empty owner, got %q", profile.Owner())
	}
	if profile.CurrentBranch() != "" {
		t.Errorf("expected empty branch, got %q", profile.CurrentBranch())
	}
	if profile.LatestCommit() != nil {
		t.Errorf("expected nil LatestCommit, got %v", profile.LatestCommit())
	}
	if profile.CommitStats() != nil {
		t.Errorf("expected nil CommitStats, got %v", profile.CommitStats())
	}
	if len(profile.Contributors()) != 0 {
		t.Errorf("expected 0 contributors, got %d", len(profile.Contributors()))
	}
	if len(profile.Tags()) != 0 {
		t.Errorf("expected 0 tags, got %d", len(profile.Tags()))
	}
	if profile.TotalFiles() != 2 {
		t.Errorf("got total files %d, want 2", profile.TotalFiles())
	}
}

func TestCollector_EmptyRepository(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	col, _ := metadata.New(disc)

	emptyRoot := filepath.Join(tempDir, "empty_repo")
	_ = os.MkdirAll(emptyRoot, 0755)

	profile, err := col.CollectPath(emptyRoot)
	if err != nil {
		t.Fatalf("CollectPath failed on empty repo: %v", err)
	}

	if profile.TotalFiles() != 0 {
		t.Errorf("expected 0 files, got %d", profile.TotalFiles())
	}
	if len(profile.Languages()) != 0 {
		t.Errorf("expected 0 languages, got %d", len(profile.Languages()))
	}
}

func TestCollector_DomainRepository(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	col, _ := metadata.New(disc)

	ws, _ := workspace.New("dom-ws", tempDir)
	proj, _ := project.New("dom-proj", ws, "my_app")
	repoRoot := filepath.Join(tempDir, "my_app", "my_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)

	repo, err := repository.New("my_repo", proj, repoRoot)
	if err != nil {
		t.Fatalf("repository.New failed: %v", err)
	}

	profile, err := col.CollectRepository(repo)
	if err != nil {
		t.Fatalf("CollectRepository failed: %v", err)
	}

	if profile.Name() != "my_repo" {
		t.Errorf("got name %q, want my_repo", profile.Name())
	}
	if profile.TotalFiles() != 1 {
		t.Errorf("got files %d, want 1", profile.TotalFiles())
	}
}

func TestCollector_SSHAndCustomGitConfigUrls(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	col, _ := metadata.New(disc)

	t.Run("SSH SCP style remote URL", func(t *testing.T) {
		repoRoot := filepath.Join(tempDir, "ssh_repo")
		gitDir := filepath.Join(repoRoot, ".git")
		_ = os.MkdirAll(gitDir, 0755)

		configContent := `[remote "origin"]
	url = git@github.com:myorg/subteam/project.git
`
		_ = os.WriteFile(filepath.Join(gitDir, "config"), []byte(configContent), 0644)
		_ = os.WriteFile(filepath.Join(repoRoot, "a.go"), []byte("package a"), 0644)

		profile, err := col.CollectPath(repoRoot)
		if err != nil {
			t.Fatalf("CollectPath failed: %v", err)
		}

		if profile.Owner() != "myorg/subteam" {
			t.Errorf("got owner %q, want myorg/subteam", profile.Owner())
		}
	})

	t.Run("Git config without remote origin leaves owner empty", func(t *testing.T) {
		repoRoot := filepath.Join(tempDir, "no_remote_repo")
		gitDir := filepath.Join(repoRoot, ".git")
		_ = os.MkdirAll(gitDir, 0755)

		configContent := `[core]
	bare = false
`
		_ = os.WriteFile(filepath.Join(gitDir, "config"), []byte(configContent), 0644)
		_ = os.WriteFile(filepath.Join(repoRoot, "a.go"), []byte("package a"), 0644)

		profile, err := col.CollectPath(repoRoot)
		if err != nil {
			t.Fatalf("CollectPath failed: %v", err)
		}

		if profile.Owner() != "" {
			t.Errorf("expected empty owner, got %q", profile.Owner())
		}
	})
}

func TestCollector_ResultImmutability(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	col, _ := metadata.New(disc)

	repoRoot := filepath.Join(tempDir, "immut_repo")
	gitDir := filepath.Join(repoRoot, ".git")
	_ = os.MkdirAll(filepath.Join(gitDir, "logs"), 0755)
	_ = os.MkdirAll(filepath.Join(gitDir, "refs", "tags"), 0755)

	_ = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644)
	_ = os.WriteFile(filepath.Join(gitDir, "refs", "tags", "v1.0.0"), []byte("1111222233334444555566667777888899990000\n"), 0644)
	logContent := "0000000000000000000000000000000000000000 1111222233334444555566667777888899990000 Alice <a@b.com> 1724490000 +0000\tcommit\n"
	_ = os.WriteFile(filepath.Join(gitDir, "logs", "HEAD"), []byte(logContent), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)

	profile, err := col.CollectPath(repoRoot)
	if err != nil {
		t.Fatalf("CollectPath failed: %v", err)
	}

	// 1. Mutate Contributors slice
	c1 := profile.Contributors()
	if len(c1) > 0 {
		c1[0] = nil
		c2 := profile.Contributors()
		if c2[0] == nil {
			t.Error("mutation of returned Contributors slice affected internal state")
		}
	}

	// 2. Mutate Tags slice
	t1 := profile.Tags()
	if len(t1) > 0 {
		t1[0] = nil
		t2 := profile.Tags()
		if t2[0] == nil {
			t.Error("mutation of returned Tags slice affected internal state")
		}
	}

	// 3. Mutate Releases slice
	r1 := profile.Releases()
	if len(r1) > 0 {
		r1[0] = nil
		r2 := profile.Releases()
		if r2[0] == nil {
			t.Error("mutation of returned Releases slice affected internal state")
		}
	}

	// 4. Mutate Languages slice
	l1 := profile.Languages()
	if len(l1) > 0 {
		l1[0] = nil
		l2 := profile.Languages()
		if l2[0] == nil {
			t.Error("mutation of returned Languages slice affected internal state")
		}
	}
}

func TestCollector_DeterminismAcrossRepeatedRuns(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	col, _ := metadata.New(disc)

	repoRoot := filepath.Join(tempDir, "det_repo")
	gitDir := filepath.Join(repoRoot, ".git")
	_ = os.MkdirAll(filepath.Join(gitDir, "logs"), 0755)
	_ = os.MkdirAll(filepath.Join(gitDir, "refs", "tags"), 0755)

	_ = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644)
	_ = os.WriteFile(filepath.Join(gitDir, "refs", "tags", "v2.0.0"), []byte("2222\n"), 0644)
	_ = os.WriteFile(filepath.Join(gitDir, "refs", "tags", "v1.0.0"), []byte("1111\n"), 0644)

	logContent := "0000 1111 User B <b@x.com> 1000 +0000\tcommit 1\n" +
		"1111 2222 User A <a@x.com> 2000 +0000\tcommit 2\n" +
		"2222 3333 User B <b@x.com> 3000 +0000\tcommit 3\n"
	_ = os.WriteFile(filepath.Join(gitDir, "logs", "HEAD"), []byte(logContent), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)

	p1, err := col.CollectPath(repoRoot)
	if err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}

	p2, err := col.CollectPath(repoRoot)
	if err != nil {
		t.Fatalf("run 2 failed: %v", err)
	}

	if p1.TotalFiles() != p2.TotalFiles() {
		t.Fatalf("file count mismatch")
	}

	contribs1 := p1.Contributors()
	contribs2 := p2.Contributors()
	if len(contribs1) != len(contribs2) {
		t.Fatalf("contributors count mismatch: %d vs %d", len(contribs1), len(contribs2))
	}
	for i := range contribs1 {
		if contribs1[i].Name() != contribs2[i].Name() || contribs1[i].CommitCount() != contribs2[i].CommitCount() {
			t.Errorf("contributor[%d] mismatch: %+v vs %+v", i, contribs1[i], contribs2[i])
		}
	}

	tags1 := p1.Tags()
	tags2 := p2.Tags()
	if len(tags1) != len(tags2) {
		t.Fatalf("tags count mismatch")
	}
	for i := range tags1 {
		if tags1[i].Name() != tags2[i].Name() {
			t.Errorf("tag[%d] mismatch: %s vs %s", i, tags1[i].Name(), tags2[i].Name())
		}
	}
}

func TestModels_MethodsAndNilSafety(t *testing.T) {
	now := time.Now()

	// Commit
	commit := metadata.NewCommit("sha123", "Raj", "raj@example.com", now, "feat: message")
	if commit.SHA() != "sha123" || commit.Author() != "Raj" || commit.Email() != "raj@example.com" || commit.Message() != "feat: message" {
		t.Errorf("Commit field mismatch")
	}
	if commit.String() == "" {
		t.Error("expected non-empty commit string")
	}
	var nilCommit *metadata.Commit
	if nilCommit.SHA() != "" || nilCommit.Author() != "" {
		t.Error("nil commit should return empty strings")
	}

	// CommitStats
	stats := metadata.NewCommitStats(10, "earliest", now.Add(-time.Hour), "latest", now)
	if stats.TotalCommits() != 10 || stats.EarliestSHA() != "earliest" || stats.LatestSHA() != "latest" || stats.TimeRange() == 0 {
		t.Errorf("CommitStats field mismatch")
	}
	var nilStats *metadata.CommitStats
	if nilStats.TotalCommits() != 0 || nilStats.EarliestSHA() != "" {
		t.Error("nil stats should return zero values")
	}

	// Contributor
	contrib := metadata.NewContributor("Raj", "raj@example.com", 5, now.Add(-time.Hour), now)
	if contrib.Name() != "Raj" || contrib.Email() != "raj@example.com" || contrib.CommitCount() != 5 {
		t.Errorf("Contributor field mismatch")
	}
	if contrib.String() == "" {
		t.Error("expected non-empty contributor string")
	}
	var nilContrib *metadata.Contributor
	if nilContrib.Name() != "" || nilContrib.CommitCount() != 0 {
		t.Error("nil contributor should return zero values")
	}

	// Tag
	tag := metadata.NewTag("v1.0.0", "sha123", "annotated", now)
	if tag.Name() != "v1.0.0" || tag.CommitSHA() != "sha123" || tag.TagType() != "annotated" {
		t.Errorf("Tag field mismatch")
	}
	if tag.String() == "" {
		t.Error("expected non-empty tag string")
	}
	var nilTag *metadata.Tag
	if nilTag.Name() != "" || nilTag.CommitSHA() != "" {
		t.Error("nil tag should return empty strings")
	}

	// Release
	rel := metadata.NewRelease("1.0.0", "v1.0.0", "sha123", false, now)
	if rel.Name() != "1.0.0" || rel.TagName() != "v1.0.0" || rel.CommitSHA() != "sha123" || rel.IsPrerelease() {
		t.Errorf("Release field mismatch")
	}
	if rel.String() == "" {
		t.Error("expected non-empty release string")
	}
	var nilRel *metadata.Release
	if nilRel.Name() != "" || nilRel.IsPrerelease() {
		t.Error("nil release should return zero values")
	}

	// Profile
	var nilProfile *metadata.Profile
	if nilProfile.Name() != "" || nilProfile.IsGit() || nilProfile.TotalFiles() != 0 {
		t.Error("nil profile should return zero values")
	}
}

func TestCollector_ScannerErrors_GitConfig(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	col, _ := metadata.New(disc)

	repoRoot := filepath.Join(tempDir, "corrupt_config_repo")
	gitDir := filepath.Join(repoRoot, ".git")
	_ = os.MkdirAll(gitDir, 0755)

	// Create oversized config file line exceeding 64KB without newline
	longLine := make([]byte, 70000)
	for i := range longLine {
		longLine[i] = 'c'
	}
	_ = os.WriteFile(filepath.Join(gitDir, "config"), longLine, 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)

	profile, err := col.CollectPath(repoRoot)
	if err != nil {
		t.Fatalf("CollectPath failed: %v", err)
	}

	var foundDiag bool
	for _, d := range profile.Diagnostics() {
		if d.Code() == "GIT_CONFIG_SCAN_ERROR" {
			foundDiag = true
			break
		}
	}
	if !foundDiag {
		t.Error("expected GIT_CONFIG_SCAN_ERROR diagnostic when .git/config scan fails")
	}
}

func TestCollector_ScannerErrors_GitLogs(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	col, _ := metadata.New(disc)

	repoRoot := filepath.Join(tempDir, "corrupt_logs_repo")
	gitDir := filepath.Join(repoRoot, ".git")
	_ = os.MkdirAll(filepath.Join(gitDir, "logs"), 0755)

	// Create oversized reflog file line exceeding 64KB without newline
	longLine := make([]byte, 70000)
	for i := range longLine {
		longLine[i] = 'l'
	}
	_ = os.WriteFile(filepath.Join(gitDir, "logs", "HEAD"), longLine, 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)

	profile, err := col.CollectPath(repoRoot)
	if err != nil {
		t.Fatalf("CollectPath failed: %v", err)
	}

	var foundDiag bool
	for _, d := range profile.Diagnostics() {
		if d.Code() == "GIT_LOGS_SCAN_ERROR" {
			foundDiag = true
			break
		}
	}
	if !foundDiag {
		t.Error("expected GIT_LOGS_SCAN_ERROR diagnostic when .git/logs/HEAD scan fails")
	}
}
