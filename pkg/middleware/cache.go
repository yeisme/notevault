package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/gin-gonic/gin"

	appcache "github.com/yeisme/notevault/pkg/cache"
)

const (
	DefaultMaxBodyBytes   = 1 << 20 // 1MB
	defaultKeyBuilderGrow = 64      // 为 key builder 预分配容量
	defaultTTL            = 30 * time.Second
)

// CacheConfig 缓存中间件配置.
type CacheConfig struct {
	Cache   *appcache.Cache                       // 必须: 业务注入的 Cache 实例
	TTL     time.Duration                         // 默认 TTL
	TTLFunc func(*gin.Context, int) time.Duration // 可选: 按请求/状态动态 TTL

	Methods     []string // 允许缓存的 HTTP 方法 (默认 GET,HEAD)
	StatusCodes []int    // 允许缓存的响应状态码 (默认 200)

	KeyFunc     func(*gin.Context) string // 生成缓存键
	Skipper     func(*gin.Context) bool   // 返回 true 跳过缓存
	VaryHeaders []string                  // 参与 Key 的 Header 列表

	RespectCacheControl bool   // 若为 true 且响应含 no-store/private 则不缓存
	BypassHeader        string // 请求头存在该 header(任意值) 则跳过缓存, 默认: X-Cache-Bypass

	MaxBodyBytes int // 缓存响应体最大字节 (0=不限制)
}

// DefaultCacheConfig 返回一份默认配置.
func DefaultCacheConfig(c *appcache.Cache) CacheConfig {
	return CacheConfig{
		Cache:               c,
		TTL:                 defaultTTL,
		Methods:             []string{"GET", "HEAD"},
		StatusCodes:         []int{http.StatusOK},
		BypassHeader:        "X-Cache-Bypass",
		MaxBodyBytes:        DefaultMaxBodyBytes,
		RespectCacheControl: true,
	}
}

// CacheMiddleware 构造缓存中间件:
//  1. 高性能: 采用内存/分布式 KV(由 cache.Cache 注入) + generic 序列化(sonic) + 可选 singleflight 防击穿
//  2. 可配置: 方法/状态码过滤、TTL/动态 TTL、Key 生成、跳过逻辑、Vary 头、最大缓存体积、绕过 Header 等
//  3. 语义友好: 支持 ETag / If-None-Match, Cache-Control: no-store/private/ max-age, X-Cache 命中标记
//  4. 安全降级: 任何缓存失败不影响主流程
//
// 使用示例:
//
//	router := gin.New()
//	c := cache.NewCache(kvStore, cache.WithLocalTTL(true))
//	router.Use(middleware.CacheMiddleware(middleware.DefaultCacheConfig(c)))
//
// 可按需覆写: cfg.KeyFunc / cfg.TTLFunc / cfg.Skipper ...
func CacheMiddleware(cfg CacheConfig) gin.HandlerFunc {
	if cfg.Cache == nil {
		panic("CacheMiddleware: Cache cannot be nil")
	}

	if len(cfg.Methods) == 0 {
		cfg.Methods = []string{"GET", "HEAD"}
	}

	if len(cfg.StatusCodes) == 0 {
		cfg.StatusCodes = []int{http.StatusOK}
	}

	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *gin.Context) string { return buildDefaultKey(c, cfg.VaryHeaders) }
	}

	if cfg.BypassHeader == "" {
		cfg.BypassHeader = "X-Cache-Bypass"
	}

	methodSet := buildMethodSet(cfg.Methods)
	statusSet := buildStatusSet(cfg.StatusCodes)

	return func(c *gin.Context) {
		if shouldBypass(c, cfg, methodSet) {
			c.Next()
			return
		}

		key := cfg.KeyFunc(c)
		if serveFromCache(c, cfg, key) {
			return
		}

		bw := &bodyCaptureWriter{ResponseWriter: c.Writer, max: cfg.MaxBodyBytes}
		c.Writer = bw
		c.Next()
		processAndStore(c, cfg, key, bw, statusSet)
	}
}

// responseCacheEntry 序列化存储结构.
type responseCacheEntry struct {
	Status   int               `json:"s"`
	Header   map[string]string `json:"h,omitempty"`
	Body     []byte            `json:"b,omitempty"`
	ETag     string            `json:"e,omitempty"`
	StoredAt int64             `json:"t"` // unix nano, 用于 Age
}

// 高性能哈希的 header key 拼接.
func buildDefaultKey(c *gin.Context, vary []string) string {
	var b strings.Builder
	b.Grow(defaultKeyBuilderGrow) // 预分配容量

	// 方法 + 路径 + 排序 query + 排序 vary headers
	// 示例: "GET:/api/v1/resource?foo=1&bar=2|hv=Accept=text/html&X-User=123"
	// 注意: query 和 headers 均排序以保证一致性
	b.WriteString(c.Request.Method)
	b.WriteByte(':')

	full := c.FullPath()
	if full == "" { // 未匹配路由时使用原始路径
		full = c.Request.URL.Path
	}

	b.WriteString(full)

	if q := c.Request.URL.Query(); len(q) > 0 { // 排序 query
		keys := make([]string, 0, len(q))
		for k := range q {
			keys = append(keys, k)
		}

		sort.Strings(keys)
		b.WriteByte('?')

		for i, k := range keys {
			if i > 0 {
				b.WriteByte('&')
			}

			b.WriteString(k)
			b.WriteByte('=')
			b.WriteString(strings.Join(q[k], ","))
		}
	}

	if len(vary) > 0 { // 参与 key 的 headers
		sort.Strings(vary)
		b.WriteString("|hv=")

		for i, h := range vary {
			if i > 0 {
				b.WriteByte('&')
			}

			b.WriteString(h)
			b.WriteByte('=')
			b.WriteString(c.GetHeader(h))
		}
	}

	return fmt.Sprintf("rc:%x", xxhash.Sum64String(b.String()))
}

// bodyCaptureWriter 包装响应写入用于捕获 body.
type bodyCaptureWriter struct {
	gin.ResponseWriter

	buf       bytes.Buffer
	max       int
	truncated bool
}

// Write 捕获响应体, 并限制最大字节数.
func (w *bodyCaptureWriter) Write(b []byte) (int, error) { // 简化条件降低复杂度
	if w.max == 0 { // 不限制
		w.buf.Write(b)
		return w.ResponseWriter.Write(b)
	}

	if w.truncated { // 已截断
		return w.ResponseWriter.Write(b)
	}

	remain := w.max - w.buf.Len()
	if remain <= 0 { // 没空间
		w.truncated = true
		return w.ResponseWriter.Write(b)
	}

	if len(b) > remain { // 部分写入
		w.buf.Write(b[:remain])
		w.truncated = true
	} else { // 全部写入
		w.buf.Write(b)
	}

	return w.ResponseWriter.Write(b)
}

// buildMethodSet 构建方法集合.
func buildMethodSet(methods []string) map[string]struct{} {
	ms := make(map[string]struct{}, len(methods))
	for _, m := range methods {
		ms[strings.ToUpper(m)] = struct{}{}
	}

	return ms
}

// buildStatusSet 构建状态码集合.
func buildStatusSet(statuses []int) map[int]struct{} {
	ss := make(map[int]struct{}, len(statuses))
	for _, s := range statuses {
		ss[s] = struct{}{}
	}

	return ss
}

// shouldBypass 检查是否应跳过缓存.
func shouldBypass(c *gin.Context, cfg CacheConfig, methodSet map[string]struct{}) bool {
	if cfg.Skipper != nil && cfg.Skipper(c) {
		return true
	}

	if _, ok := methodSet[c.Request.Method]; !ok {
		return true
	}

	if cfg.BypassHeader != "" && c.GetHeader(cfg.BypassHeader) != "" {
		return true
	}

	return false
}

// serveFromCache 尝试从缓存提供响应; 成功返回 true.
func serveFromCache(c *gin.Context, cfg CacheConfig, key string) bool {
	entry, err := appcache.Get[responseCacheEntry](c.Request.Context(), cfg.Cache, key)
	if err != nil {
		return false
	}

	inm := c.GetHeader("If-None-Match")
	if entry.ETag != "" && inm == entry.ETag { // 304 分支
		h := c.Writer.Header()
		for k, v := range entry.Header {
			h.Set(k, v)
		}

		h.Set("ETag", entry.ETag)
		age := time.Since(time.Unix(0, entry.StoredAt)).Seconds()
		h.Set("Age", fmt.Sprintf("%.0f", age))
		h.Set("X-Cache", "HIT")
		c.Status(http.StatusNotModified)
		c.Abort()

		return true
	}

	for k, v := range entry.Header {
		c.Writer.Header().Set(k, v)
	}

	if entry.ETag != "" {
		c.Writer.Header().Set("ETag", entry.ETag)
	}

	age := time.Since(time.Unix(0, entry.StoredAt)).Seconds()
	c.Writer.Header().Set("Age", fmt.Sprintf("%.0f", age))
	c.Writer.Header().Set("X-Cache", "HIT")
	c.Status(entry.Status)

	if c.Request.Method != http.MethodHead {
		_, _ = c.Writer.Write(entry.Body)
	}

	c.Abort()

	return true
}

// parseCacheControlTTL 解析 Cache-Control; 返回 (覆写TTL, 是否允许缓存).
func parseCacheControlTTL(h http.Header) (time.Duration, bool) {
	cc := h.Get("Cache-Control")
	if cc == "" {
		return 0, true
	}

	lower := strings.ToLower(cc)
	if strings.Contains(lower, "no-store") || strings.Contains(lower, "private") {
		return 0, false
	}

	if idx := strings.Index(lower, "max-age="); idx >= 0 {
		part := lower[idx+8:]
		if cidx := strings.Index(part, ","); cidx >= 0 {
			part = part[:cidx]
		}

		if d, err := time.ParseDuration(strings.TrimSpace(part) + "s"); err == nil && d > 0 {
			return d, true
		}
	}

	return 0, true
}

// processAndStore 处理响应并存储缓存.
func processAndStore(c *gin.Context, cfg CacheConfig, key string, bw *bodyCaptureWriter, statusSet map[int]struct{}) {
	status := c.Writer.Status()
	if _, ok := statusSet[status]; !ok {
		return
	}

	if bw.truncated {
		return
	}

	baseTTL := cfg.TTL
	if cfg.RespectCacheControl { // 解析 Cache-Control
		if override, ok := parseCacheControlTTL(c.Writer.Header()); !ok {
			return
		} else if override > 0 && cfg.TTLFunc == nil {
			baseTTL = override
		}
	}

	ttl := baseTTL
	if cfg.TTLFunc != nil {
		ttl = cfg.TTLFunc(c, status)
	}

	if ttl <= 0 {
		return
	}

	body := bw.buf.Bytes()
	hdr := make(map[string]string)

	for k, v := range c.Writer.Header() {
		if len(v) > 0 {
			hdr[k] = v[0]
		}
	}

	etag := c.Writer.Header().Get("ETag")
	if etag == "" {
		etag = fmt.Sprintf("\"%x\"", xxhash.Sum64(body))
		c.Writer.Header().Set("ETag", etag)
		hdr["ETag"] = etag
	}

	entry := responseCacheEntry{Status: status, Header: hdr, Body: body, ETag: etag, StoredAt: time.Now().UnixNano()}
	go func(ctx context.Context, k string, e responseCacheEntry, ttl time.Duration) {
		_ = appcache.Set(ctx, cfg.Cache, k, e, ttl)
	}(c.Request.Context(), key, entry, ttl)

	c.Writer.Header().Set("X-Cache", "MISS")
}
