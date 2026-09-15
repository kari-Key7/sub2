package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// exposedModelsSource 标识 GET /v1/models 的模型列表来自哪条分支。
type exposedModelsSource string

const (
	// exposedModelsSourceAccountMapping 分组内可调度账号 model_mapping 的 key 并集
	//（Composite 分组为各具体平台的并集）。
	exposedModelsSourceAccountMapping exposedModelsSource = "account_mapping"
	// exposedModelsSourceCustomList 分组「自定义 /v1/models 模型列表」按可用模型过滤后的结果。
	exposedModelsSourceCustomList exposedModelsSource = "custom_list"
	// exposedModelsSourcePlatformDefault 无任何账号映射时回落到的平台内置默认列表。
	exposedModelsSourcePlatformDefault exposedModelsSource = "platform_default"
)

// exposedModels 是 GET /v1/models 对某个分组实际对外暴露的模型 ID 及其来源。
type exposedModels struct {
	IDs    []string
	Source exposedModelsSource
}

// resolveExposedModels 决定 GET /v1/models 对 group 下的 API Key 在 platform
// （可能是路由强制的平台）上返回哪些模型 ID。GatewayHandler.Models 与管理端
// 「对外模型」预览共用这一份判定，二者不可能不一致。group 允许为 nil（无分组的 key）。
func resolveExposedModels(ctx context.Context, gatewayService *service.GatewayService, group *service.Group, platform string) exposedModels {
	var groupID *int64
	if group != nil {
		groupID = &group.ID
	}
	customList := group != nil && group.CustomModelsListEnabled()

	if platform == service.PlatformComposite {
		available := compositeAvailableModels(ctx, gatewayService, groupID)
		fallback := defaultModelIDsForPlatform(service.PlatformComposite)
		if customList {
			return exposedModels{
				IDs:    filterModelsByCustomList(available, fallback, group.ModelsListConfig.Models),
				Source: exposedModelsSourceCustomList,
			}
		}
		if len(available) > 0 {
			return exposedModels{IDs: available, Source: exposedModelsSourceAccountMapping}
		}
		return exposedModels{IDs: fallback, Source: exposedModelsSourcePlatformDefault}
	}

	available := gatewayService.GetAvailableModels(ctx, groupID, platform)
	if customList {
		fallback := defaultModelIDsForPlatform(platform)
		return exposedModels{
			IDs:    filterModelsByCustomList(customModelsListSource(platform, available, fallback), fallback, group.ModelsListConfig.Models),
			Source: exposedModelsSourceCustomList,
		}
	}
	if len(available) > 0 {
		return exposedModels{IDs: available, Source: exposedModelsSourceAccountMapping}
	}
	return exposedModels{IDs: fallbackModelIDsForPlatform(platform), Source: exposedModelsSourcePlatformDefault}
}

// fallbackModelIDsForPlatform 给出 writeDefaultModelsList 兜底负载里的模型 ID。
// 与 defaultModelIDsForPlatform 不同，它严格对应兜底分支：只有 OpenAI / Gemini / Grok
// 有各自的默认列表，其余平台（含 Antigravity）一律落到 Claude 列表。
func fallbackModelIDsForPlatform(platform string) []string {
	switch platform {
	case service.PlatformOpenAI:
		return openai.DefaultModelIDs()
	case service.PlatformGemini:
		return defaultModelIDsForPlatform(service.PlatformGemini)
	case service.PlatformGrok:
		return xai.DefaultModelIDs()
	default:
		return claude.DefaultModelIDs()
	}
}
