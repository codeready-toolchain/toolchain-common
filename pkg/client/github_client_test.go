package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCanIssueGitHubRequest(t *testing.T) {
	t.Run("delay threshold not expired yet", func(t *testing.T) {
		// given
		lastGitHubAPICall := time.Now().Add(-time.Second * 29) // last call was only 30 seconds ago, threshold is 1 minute

		// when
		ok := CanIssueGitHubRequest(lastGitHubAPICall)

		// then
		require.False(t, ok) // we should wait
	})

	t.Run("delay threshold expired", func(t *testing.T) {
		// given
		lastGitHubAPICall := time.Now().Add(-time.Minute * 1) // last call was 1 minute, delay expired

		// when
		ok := CanIssueGitHubRequest(lastGitHubAPICall)

		// then
		require.True(t, ok) // ok to call
	})

	t.Run("first time we call github api", func(t *testing.T) {
		// given
		var lastGitHubAPICall time.Time

		// when
		ok := CanIssueGitHubRequest(lastGitHubAPICall)

		// then
		require.True(t, ok) // ok to call
	})
}
