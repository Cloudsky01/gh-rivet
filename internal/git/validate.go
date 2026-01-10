package git

import (
	"fmt"
	"regexp"
)

// ValidateRepositoryFormat validates that a repository string is in the correct owner/repo format
func ValidateRepositoryFormat(repo string) error {
	pattern := regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)

	if !pattern.MatchString(repo) {
		return fmt.Errorf("invalid repository format: %q - expected format: owner/repo", repo)
	}

	return nil
}
