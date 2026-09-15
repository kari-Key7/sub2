package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type exposedModelsGroupRepoStub struct {
	service.GroupRepository

	groups []service.Group
	err    error
}

func (s *exposedModelsGroupRepoStub) ListActive(ctx context.Context) ([]service.Group, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make([]service.Group, len(s.groups))
	copy(out, s.groups)
	return out, nil
}

type exposedModelsResponseForTest struct {
	Code int `json:"code"`
	Data struct {
		Groups []struct {
			ID             int64    `json:"id"`
			Name           string   `json:"name"`
			Platform       string   `json:"platform"`
			RateMultiplier float64  `json:"rate_multiplier"`
			Source         string   `json:"source"`
			Models         []string `json:"models"`
		} `json:"groups"`
	} `json:"data"`
}

func openAIMappingAccount(models ...string) service.Account {
	mapping := make(map[string]any, len(models))
	for _, m := range models {
		mapping[m] = m
	}
	return service.Account{
		ID:          1,
		Platform:    service.PlatformOpenAI,
		Credentials: map[string]any{"model_mapping": mapping},
	}
}

// gatewayModelIDsForGroup runs the real GET /v1/models handler for a key in the
// given group and returns the advertised model IDs.
func gatewayModelIDsForGroup(t *testing.T, h *GatewayHandler, group *service.Group) []string {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{Group: group})

	h.Models(c)
	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return modelIDsForTest(got.Data)
}

func TestResolveExposedModels_AccountMappingSource(t *testing.T) {
	groupID := int64(31)
	h := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{
		byGroup: map[int64][]service.Account{
			groupID: {openAIMappingAccount("gpt-5.5", "gpt-5.4")},
		},
	})
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI}

	got := resolveExposedModels(context.Background(), h.gatewayService, group, group.Platform)

	require.Equal(t, exposedModelsSourceAccountMapping, got.Source)
	require.Equal(t, []string{"gpt-5.4", "gpt-5.5"}, got.IDs)
}

func TestResolveExposedModels_CustomListSource(t *testing.T) {
	groupID := int64(32)
	h := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{
		byGroup: map[int64][]service.Account{
			groupID: {openAIMappingAccount("gpt-5.5", "gpt-5.4", "legacy-gpt-2024")},
		},
	})
	group := &service.Group{
		ID:       groupID,
		Platform: service.PlatformOpenAI,
		ModelsListConfig: service.GroupModelsListConfig{
			Enabled: true,
			Models:  []string{"gpt-5.5", "missing-model", "gpt-5.4"},
		},
	}

	got := resolveExposedModels(context.Background(), h.gatewayService, group, group.Platform)

	require.Equal(t, exposedModelsSourceCustomList, got.Source)
	require.Equal(t, []string{"gpt-5.5", "gpt-5.4"}, got.IDs)
}

func TestResolveExposedModels_PlatformDefaultSource(t *testing.T) {
	h := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{})
	group := &service.Group{ID: 33, Platform: service.PlatformAnthropic}

	got := resolveExposedModels(context.Background(), h.gatewayService, group, group.Platform)

	require.Equal(t, exposedModelsSourcePlatformDefault, got.Source)
	require.Equal(t, claude.DefaultModelIDs(), got.IDs)
}

// Scenario: the admin preview must advertise exactly what GET /v1/models returns
// for a key in the same group, across every branch the gateway handler can take.
func TestExposedModelsHandler_ListMatchesGatewayModels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groups := []service.Group{
		{ID: 41, Name: "openai-mapped", Platform: service.PlatformOpenAI, RateMultiplier: 1},
		{
			ID: 42, Name: "openai-custom", Platform: service.PlatformOpenAI, RateMultiplier: 1.5,
			ModelsListConfig: service.GroupModelsListConfig{Enabled: true, Models: []string{"gpt-5.5"}},
		},
		{ID: 43, Name: "anthropic-empty", Platform: service.PlatformAnthropic, RateMultiplier: 1},
		{ID: 44, Name: "antigravity-empty", Platform: service.PlatformAntigravity, RateMultiplier: 1},
		{ID: 45, Name: "composite", Platform: service.PlatformComposite, RateMultiplier: 2},
		{ID: 46, Name: "openai-empty", Platform: service.PlatformOpenAI, RateMultiplier: 1},
		{ID: 47, Name: "gemini-empty", Platform: service.PlatformGemini, RateMultiplier: 1},
		{ID: 48, Name: "grok-empty", Platform: service.PlatformGrok, RateMultiplier: 1},
		{ID: 49, Name: "composite-empty", Platform: service.PlatformComposite, RateMultiplier: 1},
	}
	accountRepo := &gatewayModelsAccountRepoStub{
		byGroup: map[int64][]service.Account{
			41: {openAIMappingAccount("gpt-5.5", "gpt-5.4")},
			42: {openAIMappingAccount("gpt-5.5", "gpt-5.4")},
			45: {
				openAIMappingAccount("gpt-5.5"),
				{
					ID:          2,
					Platform:    service.PlatformAnthropic,
					Credentials: map[string]any{"model_mapping": map[string]any{"claude-opus-4-6": "claude-opus-4-6"}},
				},
			},
		},
	}
	gateway := newGatewayModelsHandlerForTest(accountRepo)
	admin := NewExposedModelsHandler(gateway.gatewayService, &exposedModelsGroupRepoStub{groups: groups})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/exposed-models", nil)
	admin.List(c)
	require.Equal(t, http.StatusOK, rec.Code)

	var got exposedModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Data.Groups, len(groups))

	wantSources := map[int64]string{
		41: "account_mapping",
		42: "custom_list",
		43: "platform_default",
		44: "platform_default",
		45: "account_mapping",
		46: "platform_default",
		47: "platform_default",
		48: "platform_default",
		49: "platform_default",
	}
	for i := range groups {
		group := groups[i]
		entry := got.Data.Groups[i]
		require.Equal(t, group.ID, entry.ID)
		require.Equal(t, group.Name, entry.Name)
		require.Equal(t, group.Platform, entry.Platform)
		require.Equal(t, group.RateMultiplier, entry.RateMultiplier)
		require.Equal(t, wantSources[group.ID], entry.Source, "group %s", group.Name)
		require.Equal(t, gatewayModelIDsForGroup(t, gateway, &group), entry.Models, "group %s", group.Name)
		require.NotEmpty(t, entry.Models, "group %s", group.Name)
	}
}

func TestExposedModelsHandler_List_RepoErrorReturns500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gateway := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{})
	admin := NewExposedModelsHandler(gateway.gatewayService, &exposedModelsGroupRepoStub{err: errors.New("db down")})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/exposed-models", nil)
	admin.List(c)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
