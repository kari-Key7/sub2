package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newBotBlockRouter(cfg config.BotBlockConfig) *gin.Engine {
	r := gin.New()
	r.Use(BotBlock(cfg))
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/v1/models", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func doBotBlockRequest(r *gin.Engine, path, ua string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestBotBlock_BlocksKnownAICrawlers(t *testing.T) {
	r := newBotBlockRouter(config.BotBlockConfig{Enabled: true})

	cases := []string{
		"Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; GPTBot/1.2; +https://openai.com/gptbot)",
		"Mozilla/5.0 (compatible; ChatGPT-User/1.0; +https://openai.com/bot)",
		"Mozilla/5.0 (compatible; OAI-SearchBot/1.0; +https://openai.com/searchbot)",
		"Mozilla/5.0 (compatible; ClaudeBot/1.0; +claudebot@anthropic.com)",
		"Mozilla/5.0 (compatible; Claude-User/1.0; +Claude-User@anthropic.com)",
		"Mozilla/5.0 (compatible; Claude-SearchBot/1.0)",
		"Mozilla/5.0 (compatible; anthropic-ai/1.0)",
		"Mozilla/5.0 (compatible; PerplexityBot/1.0; +https://perplexity.ai/perplexitybot)",
		"Mozilla/5.0 (compatible; Perplexity-User/1.0)",
		"Mozilla/5.0 (compatible; Google-Extended)",
		"Mozilla/5.0 (compatible; CCBot/2.0; +https://commoncrawl.org/faq/)",
		"Mozilla/5.0 (Linux; Android 5.0) AppleWebKit/537.36 (KHTML, like Gecko) Mobile Safari/537.36 (compatible; Bytespider; spider-feedback@bytedance.com)",
		"Mozilla/5.0 (compatible; Amazonbot/0.1; +https://developer.amazon.com/support/amazonbot)",
		"meta-externalagent/1.1 (+https://developers.facebook.com/docs/sharing/webmasters/crawler)",
		"Mozilla/5.0 (compatible; Applebot-Extended/0.1)",
		"Mozilla/5.0 (compatible; cohere-ai/1.0)",
		"Mozilla/5.0 (compatible; Diffbot/1.0)",
		"Mozilla/5.0 (compatible; YouBot/1.0)",
		"Mozilla/5.0 (compatible; DuckAssistBot/1.0)",
		"Mozilla/5.0 (compatible; MistralAI-User/1.0)",
		"gptbot/1.0", // 大小写不敏感
	}
	for _, ua := range cases {
		w := doBotBlockRequest(r, "/", ua)
		assert.Equal(t, http.StatusForbidden, w.Code, "UA should be blocked: %s", ua)
		assert.Equal(t, "Forbidden", w.Body.String(), "body should be generic for: %s", ua)
	}
}

func TestBotBlock_AllowsNormalClients(t *testing.T) {
	r := newBotBlockRouter(config.BotBlockConfig{Enabled: true})

	cases := []string{
		"", // 无 UA
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
		"curl/8.4.0",
		"claude-cli/1.0.0 (external, cli)", // Claude Code 客户端，不是爬虫
		"OpenAI/Python 1.30.0",
		"codex_cli_rs/0.20.0",
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", // 普通搜索引擎不在名单
		"python-requests/2.31",
	}
	for _, ua := range cases {
		w := doBotBlockRequest(r, "/", ua)
		assert.Equal(t, http.StatusOK, w.Code, "UA should pass: %q", ua)
	}
}

func TestBotBlock_AppliesToAPIPathsToo(t *testing.T) {
	r := newBotBlockRouter(config.BotBlockConfig{Enabled: true})
	w := doBotBlockRequest(r, "/v1/models", "Mozilla/5.0 (compatible; GPTBot/1.0)")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestBotBlock_Disabled(t *testing.T) {
	r := newBotBlockRouter(config.BotBlockConfig{Enabled: false})
	w := doBotBlockRequest(r, "/", "Mozilla/5.0 (compatible; GPTBot/1.0)")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBotBlock_ExtraUserAgents(t *testing.T) {
	r := newBotBlockRouter(config.BotBlockConfig{
		Enabled:         true,
		ExtraUserAgents: []string{"MyPrivateScanner", "  ", ""},
	})
	assert.Equal(t, http.StatusForbidden, doBotBlockRequest(r, "/", "myprivatescanner/2.0").Code)
	assert.Equal(t, http.StatusOK, doBotBlockRequest(r, "/", "curl/8.4.0").Code)
}
