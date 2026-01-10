package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractRepoFromURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "HTTPS URL with .git suffix",
			url:  "https://github.com/octocat/Hello-World.git",
			want: "octocat/Hello-World",
		},
		{
			name: "HTTPS URL without .git suffix",
			url:  "https://github.com/octocat/Hello-World",
			want: "octocat/Hello-World",
		},
		{
			name: "HTTPS URL with trailing slash",
			url:  "https://github.com/octocat/Hello-World/",
			want: "octocat/Hello-World",
		},
		{
			name: "SSH URL with .git suffix",
			url:  "git@github.com:octocat/Hello-World.git",
			want: "octocat/Hello-World",
		},
		{
			name: "SSH URL without .git suffix",
			url:  "git@github.com:octocat/Hello-World",
			want: "octocat/Hello-World",
		},
		{
			name: "HTTP URL",
			url:  "http://github.com/owner/repo-name.git",
			want: "owner/repo-name",
		},
		{
			name: "Repo with dots",
			url:  "https://github.com/owner/my.repo.git",
			want: "owner/my.repo",
		},
		{
			name: "Repo with underscores",
			url:  "https://github.com/my_owner/my_repo",
			want: "my_owner/my_repo",
		},
		{
			name: "Invalid URL - missing path",
			url:  "https://github.com/",
			want: "",
		},
		{
			name: "Invalid URL - single component",
			url:  "https://github.com/octocat",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRepoFromURL(tt.url)
			if got != tt.want {
				t.Errorf("extractRepoFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestIsValidRepoFormat(t *testing.T) {
	tests := []struct {
		name string
		repo string
		want bool
	}{
		{
			name: "Valid format - simple",
			repo: "owner/repo",
			want: true,
		},
		{
			name: "Valid format - with dashes",
			repo: "my-owner/my-repo",
			want: true,
		},
		{
			name: "Valid format - with underscores",
			repo: "my_owner/my_repo",
			want: true,
		},
		{
			name: "Valid format - with dots",
			repo: "owner.io/my.repo",
			want: true,
		},
		{
			name: "Valid format - mixed characters",
			repo: "octocat/Hello-World_2",
			want: true,
		},
		{
			name: "Invalid format - no slash",
			repo: "ownerrepo",
			want: false,
		},
		{
			name: "Invalid format - multiple slashes",
			repo: "owner/repo/name",
			want: false,
		},
		{
			name: "Invalid format - empty parts",
			repo: "/repo",
			want: false,
		},
		{
			name: "Invalid format - special characters",
			repo: "owner@/repo",
			want: false,
		},
		{
			name: "Invalid format - spaces",
			repo: "owner /repo",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidRepoFormat(tt.repo)
			if got != tt.want {
				t.Errorf("isValidRepoFormat(%q) = %v, want %v", tt.repo, got, tt.want)
			}
		})
	}
}

func TestParseGitConfig(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		want          string
		wantErr       bool
	}{
		{
			name: "Valid HTTPS origin",
			configContent: `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/octocat/Hello-World.git
	fetch = +refs/heads/*:refs/remotes/origin/*`,
			want: "octocat/Hello-World",
		},
		{
			name: "Valid SSH origin",
			configContent: `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = git@github.com:octocat/Hello-World.git
	fetch = +refs/heads/*:refs/remotes/origin/*`,
			want: "octocat/Hello-World",
		},
		{
			name: "Multiple remotes - origin comes first",
			configContent: `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/octocat/Hello-World
	fetch = +refs/heads/*:refs/remotes/origin/*
[remote "upstream"]
	url = https://github.com/other/repo.git
	fetch = +refs/heads/*:refs/remotes/upstream/*`,
			want: "octocat/Hello-World",
		},
		{
			name: "No origin remote",
			configContent: `[core]
	repositoryformatversion = 0
[remote "upstream"]
	url = https://github.com/octocat/Hello-World.git`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpfile, err := os.CreateTemp("", "git-config-test-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.WriteString(tt.configContent); err != nil {
				t.Fatal(err)
			}
			tmpfile.Close()

			got, err := parseGitConfig(tmpfile.Name())
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGitConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("parseGitConfig() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindGitConfig(t *testing.T) {
	// Create a temporary directory structure
	tmpdir := t.TempDir()
	gitDir := filepath.Join(tmpdir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(gitDir, "config")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to the temporary directory and test
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(tmpdir); err != nil {
		t.Fatal(err)
	}

	found, err := findGitConfig()
	if err != nil {
		t.Errorf("findGitConfig() error = %v", err)
		return
	}

	// Normalize both paths in case one uses symlinks
	expectedAbs, _ := filepath.EvalSymlinks(configPath)
	foundAbs, _ := filepath.EvalSymlinks(found)

	if foundAbs != expectedAbs {
		t.Errorf("findGitConfig() = %q, want %q", foundAbs, expectedAbs)
	}
}
