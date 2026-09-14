package middleware

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

// defaultBlockedUserAgents 是已知 AI 爬虫 / AI 代抓取（用户把网址贴给 AI 后由 AI 侧发起）的
// User-Agent 标识片段，匹配时大小写不敏感。普通搜索引擎与 API 客户端（claude-cli、
// codex_cli_rs、OpenAI SDK 等）不在名单内。
var defaultBlockedUserAgents = []string{
	// OpenAI
	"gptbot",
	"chatgpt-user",
	"oai-searchbot",
	// Anthropic
	"claudebot",
	"claude-user",
	"claude-searchbot",
	"claude-web",
	"anthropic-ai",
	// Perplexity
	"perplexitybot",
	"perplexity-user",
	// Google AI training / Gemini
	"google-extended",
	"google-cloudvertexbot",
	// Meta
	"meta-externalagent",
	"meta-externalfetcher",
	"facebookbot",
	// Apple AI
	"applebot-extended",
	// ByteDance / others
	"bytespider",
	"amazonbot",
	"ccbot",
	"cohere-ai",
	"diffbot",
	"youbot",
	"duckassistbot",
	"mistralai-user",
	"ai2bot",
	"omgili",
	"img2dataset",
	"timpibot",
	"petalbot",
	"iaskbot",
	"kangaroo bot",
	"webzio-extended",
	"scrapy",
}

// BotBlock 拦截命中名单的 User-Agent，返回不带任何结构信息的 403。
// 目的是防止 AI 抓取工具直接读取站点内容；它挡不住普通浏览器 UA 的访问，
// 只是作为去指纹改造之外的补充层。
func BotBlock(cfg config.BotBlockConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}

	tokens := make([]string, 0, len(defaultBlockedUserAgents)+len(cfg.ExtraUserAgents))
	tokens = append(tokens, defaultBlockedUserAgents...)
	for _, extra := range cfg.ExtraUserAgents {
		extra = strings.ToLower(strings.TrimSpace(extra))
		if extra == "" {
			continue
		}
		tokens = append(tokens, extra)
	}

	return func(c *gin.Context) {
		ua := strings.ToLower(c.Request.UserAgent())
		if ua == "" {
			c.Next()
			return
		}
		for _, token := range tokens {
			if strings.Contains(ua, token) {
				c.Header("Cache-Control", "no-store")
				c.String(http.StatusForbidden, "Forbidden")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
