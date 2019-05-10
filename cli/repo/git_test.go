package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vsco/dcdr/config"
)

func TestGitEnabled (t *testing.T) {
	git := New(&config.Config{});

	assert.Equal(t, false, git.Enabled())
	assert.Equal(t, false, git.PushEnabled())

	git.Config.Git.RepoPath = "/foo/bar"

	assert.Equal(t, true, git.Enabled())
	assert.Equal(t, false, git.PushEnabled())

	git.Config.Git.RepoURL = "git@git.code:foo/bar.git"

	assert.Equal(t, true, git.Enabled())
	assert.Equal(t, true, git.PushEnabled())
}

func TestPull (t *testing.T) {
	git := New(&config.Config{});
	err := git.Pull()

	// Pull returns immediately with nil if not PushEnabled()
	assert.Equal(t, nil, err)
}

func TestPush (t *testing.T) {
	git := New(&config.Config{});
	err := git.Push()

	// Push returns immediately with nil if not PushEnabled()
	assert.Equal(t, nil, err)
}
