package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ExposedModelsHandler 管理端「对外模型」：按分组预览 GET /v1/models 实际返回的模型列表。
type ExposedModelsHandler struct {
	gatewayService *service.GatewayService
	groupRepo      service.GroupRepository
}

// NewExposedModelsHandler 创建对外模型预览处理器。
func NewExposedModelsHandler(gatewayService *service.GatewayService, groupRepo service.GroupRepository) *ExposedModelsHandler {
	return &ExposedModelsHandler{
		gatewayService: gatewayService,
		groupRepo:      groupRepo,
	}
}

// exposedModelsGroupDTO 单个分组对外暴露的模型列表。
type exposedModelsGroupDTO struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Platform       string  `json:"platform"`
	RateMultiplier float64 `json:"rate_multiplier"`
	// Source 列表来源：account_mapping / custom_list / platform_default，见 exposedModelsSource。
	Source string   `json:"source"`
	Models []string `json:"models"`
}

type exposedModelsResponse struct {
	Groups []exposedModelsGroupDTO `json:"groups"`
}

// List 返回每个活跃分组下 GET /v1/models 会返回的模型 ID（口径与网关完全一致）。
// GET /api/v1/admin/exposed-models
func (h *ExposedModelsHandler) List(c *gin.Context) {
	groups, err := h.groupRepo.ListActive(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]exposedModelsGroupDTO, 0, len(groups))
	for i := range groups {
		g := &groups[i]
		exposed := resolveExposedModels(c.Request.Context(), h.gatewayService, g, g.Platform)
		models := exposed.IDs
		if models == nil {
			models = []string{}
		}
		out = append(out, exposedModelsGroupDTO{
			ID:             g.ID,
			Name:           g.Name,
			Platform:       g.Platform,
			RateMultiplier: g.RateMultiplier,
			Source:         string(exposed.Source),
			Models:         models,
		})
	}

	response.Success(c, exposedModelsResponse{Groups: out})
}
