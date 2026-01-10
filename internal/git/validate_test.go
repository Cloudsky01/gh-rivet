package git

import (
	"testing"
)

func TestValidateRepositoryFormat(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		wantErr bool
	}{
		{
			name:    "Valid format - simple",
			repo:    "owner/repo",
			wantErr: false,
		},
		{
			name:    "Valid format - with dashes",
			repo:    "my-owner/my-repo",
			wantErr: false,
		},
		{
			name:    "Valid format - with underscores",
			repo:    "my_owner/my_repo",
			wantErr: false,
		},
		{
			name:    "Valid format - with dots",
			repo:    "owner.io/my.repo",
			wantErr: false,
		},
		{
			name:    "Valid format - numbers",
			repo:    "owner123/repo456",
			wantErr: false,
		},
		{
			name:    "Invalid format - no slash",
			repo:    "ownerrepo",
			wantErr: true,
		},
		{
			name:    "Invalid format - multiple slashes",
			repo:    "owner/repo/name",
			wantErr: true,
		},
		{
			name:    "Invalid format - empty owner",
			repo:    "/repo",
			wantErr: true,
		},
		{
			name:    "Invalid format - empty repo",
			repo:    "owner/",
			wantErr: true,
		},
		{
			name:    "Invalid format - special characters",
			repo:    "owner@/repo",
			wantErr: true,
		},
		{
			name:    "Invalid format - spaces",
			repo:    "owner /repo",
			wantErr: true,
		},
		{
			name:    "Invalid format - empty string",
			repo:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepositoryFormat(tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRepositoryFormat(%q) error = %v, wantErr %v", tt.repo, err, tt.wantErr)
			}
		})
	}
}
