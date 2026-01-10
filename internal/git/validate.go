package git

import (
	"fmt"
	"regexp"
)

// repository format should be owner/repo
var RepositoryFormatRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)

func ValidateRepositoryFormat(repo string) error {
	if !RepositoryFormatRegex.MatchString(repo) {
		return fmt.Errorf("invalid repository format: %q - expected format: owner/repo", repo)
	}

	return nil
}
