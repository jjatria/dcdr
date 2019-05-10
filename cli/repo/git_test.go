package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vsco/dcdr/config"
)

func TestGitEnabled (t *testing.T) {
	git := New(&config.Config{});

	assert.Equal(t, false, git.Enabled())

	git.Config.Git.RepoPath = "/foo/bar"

	assert.Equal(t, true, git.Enabled())
}
