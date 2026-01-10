package git

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DetectRepository attempts to detect the GitHub repository from .git/config
// Returns the repository in owner/repo format, or an error if detection fails
func DetectRepository() (string, error) {
	// Try to find .git/config in current directory or parent directories
	gitConfigPath, err := findGitConfig()
	if err != nil {
		return "", err
	}

	// Read and parse the config file
	repo, err := parseGitConfig(gitConfigPath)
	if err != nil {
		return "", err
	}

	return repo, nil
}

// findGitConfig locates the .git/config file by searching upward from current directory
func findGitConfig() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Search up to 10 levels deep
	for range 10 {
		gitDir := filepath.Join(cwd, ".git")
		configPath := filepath.Join(gitDir, "config")

		if info, err := os.Stat(configPath); err == nil && !info.IsDir() {
			return configPath, nil
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			// Reached filesystem root
			break
		}
		cwd = parent
	}

	return "", fmt.Errorf("no .git/config found - not in a git repository")
}

// parseGitConfig reads the git config file and extracts the GitHub repository
func parseGitConfig(configPath string) (string, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read git config: %w", err)
	}

	// Look for origin remote URL
	lines := strings.Split(string(content), "\n")
	var inOriginSection bool
	var url string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for [remote "origin"] section
		if strings.HasPrefix(trimmed, "[remote") && strings.Contains(trimmed, "origin") {
			inOriginSection = true
			continue
		}

		// Check for other sections (end of origin section)
		if inOriginSection && strings.HasPrefix(trimmed, "[") && !strings.Contains(trimmed, "origin") {
			inOriginSection = false
		}

		// Extract URL from origin section
		if inOriginSection && strings.HasPrefix(trimmed, "url") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				url = strings.TrimSpace(parts[1])
				break
			}
		}
	}

	if url == "" {
		return "", fmt.Errorf("no origin remote found in git config")
	}

	// Parse various GitHub URL formats
	repo := extractRepoFromURL(url)
	if repo == "" {
		return "", fmt.Errorf("failed to extract owner/repo from URL: %s", url)
	}

	return repo, nil
}

// extractRepoFromURL converts various GitHub URL formats to owner/repo format
// Handles:
//   - https://github.com/owner/repo
//   - https://github.com/owner/repo.git
//   - git@github.com:owner/repo
//   - git@github.com:owner/repo.git
func extractRepoFromURL(url string) string {
	// Handle HTTPS URLs
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		// Remove protocol and domain: https://github.com/owner/repo.git -> owner/repo.git
		if idx := strings.Index(url, "github.com/"); idx != -1 {
			repo := url[idx+len("github.com/"):]
			// Remove .git suffix if present
			repo = strings.TrimSuffix(repo, ".git")
			// Remove trailing slash
			repo = strings.TrimSuffix(repo, "/")
			if isValidRepoFormat(repo) {
				return repo
			}
		}
	}

	// Handle SSH URLs: git@github.com:owner/repo.git -> owner/repo
	if after, ok := strings.CutPrefix(url, "git@github.com:"); ok {
		repo := after
		// Remove .git suffix if present
		repo = strings.TrimSuffix(repo, ".git")
		if isValidRepoFormat(repo) {
			return repo
		}
	}

	return ""
}

// isValidRepoFormat checks if string is in owner/repo format
func isValidRepoFormat(repo string) bool {
	// Match: owner/repo where both parts contain only alphanumeric, hyphen, underscore, dot
	pattern := regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)
	return pattern.MatchString(repo)
}
