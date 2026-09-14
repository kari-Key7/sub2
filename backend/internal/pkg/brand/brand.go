// Package brand 集中定义对外可见的产品标识。
//
// 所有会出现在 HTTP 响应、页面、邮件、2FA 二维码、导出文件等外部可观测位置的
// 品牌名/标识都应从这里引用，避免散落硬编码。纯内部使用的键（Redis 前缀、
// 数据库表名、临时文件名等）不在此范围内。
package brand

const (
	// Name 产品名，作为站点名未配置时的回退值，也用于 TOTP issuer、邮件、订单描述等。
	Name = "KimAI"

	// Tagline 浏览器标题后缀。
	Tagline = "AI API Gateway"

	// Subtitle 站点副标题默认值（site_subtitle 为空时使用）。
	Subtitle = "AI API Gateway"

	// AdminWSProtocol 管理端 WebSocket 子协议名，需与前端 api/admin/ops.ts 保持一致。
	AdminWSProtocol = "kimai-admin"

	// ExportDataType 账号/代理导入导出文件的 type 字段，需与前端 ImportDataModal 保持一致。
	ExportDataType = "kimai-data"

	// KeyBillingObject /v1/{slug}/billing 响应中的 object 字段。
	KeyBillingObject = "kimai.key_billing"

	// KeyBillingPath API Key 计费信息端点路径（不含 /v1 前缀之外的部分）。
	KeyBillingPath = "/v1/kimai/billing"

	// GrokClientToolCacheHeader Grok 客户端工具缓存 opt-in 请求头。
	GrokClientToolCacheHeader = "X-KimAI-Grok-Client-Tool-Cache"
)
