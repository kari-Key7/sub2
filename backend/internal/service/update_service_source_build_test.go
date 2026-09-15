//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// countingGitHubClientStub 记录是否访问过 GitHub，用于证明源码构建根本不会去查上游。
type countingGitHubClientStub struct {
	updateServiceGitHubClientStub
	latestCalls int
	recentCalls int
}

func (s *countingGitHubClientStub) FetchLatestRelease(ctx context.Context, repo string) (*GitHubRelease, error) {
	s.latestCalls++
	return s.updateServiceGitHubClientStub.FetchLatestRelease(ctx, repo)
}

func (s *countingGitHubClientStub) FetchRecentReleases(ctx context.Context, repo string, perPage int) ([]*GitHubRelease, error) {
	s.recentCalls++
	return s.updateServiceGitHubClientStub.FetchRecentReleases(ctx, repo, perPage)
}

func newSourceBuildUpdateService(client GitHubReleaseClient) *UpdateService {
	return NewUpdateService(&updateServiceCacheStub{}, client, "0.2.5", "source")
}

// 场景：自建镜像（BuildType=source）上游发布了更高版本。
// 在线更新会用官方二进制覆盖本 fork 的定制版本，所以必须整体关闭，且不联网查询。
func TestUpdateServiceSourceBuild_CheckUpdateReportsNoUpdateWithoutCallingGitHub(t *testing.T) {
	client := &countingGitHubClientStub{
		updateServiceGitHubClientStub: updateServiceGitHubClientStub{
			release: &GitHubRelease{TagName: "v9.9.9", Name: "v9.9.9"},
		},
	}
	svc := newSourceBuildUpdateService(client)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.False(t, info.HasUpdate)
	require.Equal(t, "0.2.5", info.CurrentVersion)
	require.Equal(t, "0.2.5", info.LatestVersion)
	require.Equal(t, "source", info.BuildType)
	require.Zero(t, client.latestCalls, "source builds must not query upstream releases")
}

func TestUpdateServiceSourceBuild_MutatingOperationsAreRefused(t *testing.T) {
	client := &countingGitHubClientStub{
		updateServiceGitHubClientStub: updateServiceGitHubClientStub{
			release:        &GitHubRelease{TagName: "v9.9.9", Name: "v9.9.9"},
			recentReleases: []*GitHubRelease{{TagName: "v0.2.4", Name: "v0.2.4"}},
		},
	}
	svc := newSourceBuildUpdateService(client)
	ctx := context.Background()

	require.True(t, errors.Is(svc.PerformUpdate(ctx), ErrOnlineUpdateDisabled))
	require.True(t, errors.Is(svc.Rollback(), ErrOnlineUpdateDisabled))
	require.True(t, errors.Is(svc.RollbackToVersion(ctx, "0.2.4"), ErrOnlineUpdateDisabled))

	versions, err := svc.ListRollbackVersions(ctx)
	require.NoError(t, err)
	require.Empty(t, versions)

	require.Zero(t, client.latestCalls)
	require.Zero(t, client.recentCalls)
}

// 对照：release 构建保持原有行为，仍会查询上游。
func TestUpdateServiceReleaseBuild_StillChecksGitHub(t *testing.T) {
	client := &countingGitHubClientStub{
		updateServiceGitHubClientStub: updateServiceGitHubClientStub{
			release: &GitHubRelease{TagName: "v9.9.9", Name: "v9.9.9"},
		},
	}
	svc := NewUpdateService(&updateServiceCacheStub{}, client, "0.2.5", "release")

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.True(t, info.HasUpdate)
	require.Equal(t, 1, client.latestCalls)
}
