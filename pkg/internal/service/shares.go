package service

import (
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/oklog/ulid"

	ctxPkg "github.com/yeisme/notevault/pkg/context"
	"github.com/yeisme/notevault/pkg/internal/model"
	"github.com/yeisme/notevault/pkg/internal/storage/db"
	"github.com/yeisme/notevault/pkg/internal/storage/kv"
	"github.com/yeisme/notevault/pkg/internal/storage/s3"
	"github.com/yeisme/notevault/pkg/internal/types"
	nlog "github.com/yeisme/notevault/pkg/log"
)

// 全局单例的 ULID 熵源，使用单调递增策略，确保同一毫秒内生成的 ULID 具有排序稳定性。
var ulidEntropy = ulid.Monotonic(crand.Reader, 0)

// ShareService 负责分享相关业务（默认基于 KV 存储，可平滑切换到 DB 实现）.
type ShareService struct {
	dbc *db.Client
	kvc *kv.Client
	s3c *s3.Client
}

// NewShareService 创建并返回一个新的 ShareService 实例.
func NewShareService(c context.Context) *ShareService {
	svc := &ShareService{
		dbc: ctxPkg.GetDBClient(c),
		kvc: ctxPkg.GetKVClient(c),
		s3c: ctxPkg.GetS3Client(c),
	}

	if svc.dbc == nil {
		nlog.Logger().Warn().Msg("DB client not initialized, ShareService features limited")
	}

	if svc.s3c == nil {
		nlog.Logger().Warn().Msg("S3 client not initialized, download URL features will be limited")
	}

	return svc
}

// CreateShare 创建一个新的分享，返回分享信息.
func (s *ShareService) CreateShare(ctx context.Context, user string, req *types.CreateShareRequest) (*types.CreateShareResponse, error) {
	if user == "" {
		return nil, fmt.Errorf("user is required")
	}

	if req == nil || len(req.ObjectKeys) == 0 {
		return nil, fmt.Errorf("object_keys is required")
	}

	if s.dbc == nil || s.dbc.GetDB() == nil {
		return nil, errors.New("db not initialized")
	}

	now := time.Now().UTC()

	var (
		expire *time.Time
		ttl    time.Duration
	)

	if req.ExpireDays > 0 {
		e := now.Add(time.Duration(req.ExpireDays) * 24 * time.Hour)
		expire = &e

		ttl = max(time.Until(e), 0)
	}

	// 使用 ULID 生成 shareID，保证按时间排序且唯一；保留 "sh_" 前缀以兼容历史格式
	shareID := newShareID(now)
	rec := &shareRecord{
		ShareID:       shareID,
		Owner:         user,
		ObjectKeys:    req.ObjectKeys,
		CreatedAt:     now,
		ExpireAt:      expire,
		AllowDownload: req.AllowDownload,
		PasswordHash:  hashPassword(req.Password),
		Permissions: types.SharePermissions{
			// 默认允许匿名访问（若设置了密码则按密码校验）
			AllowAnonymous: req.Password == "",
			Users:          []string{user}, // 创建者默认在用户列表中
		},
	}

	// 写 DB
	dbRec, err := model.FromRecord((*model.ShareRecord)(rec))
	if err != nil {
		return nil, err
	}

	if err := s.dbc.GetDB().Create(dbRec).Error; err != nil {
		return nil, fmt.Errorf("create share: %w", err)
	}

	// 轻缓存（可选）：写入 share 缓存
	_ = s.cacheShare(ctx, rec, ttl)

	return &types.CreateShareResponse{Share: rec.toInfo()}, nil
}

// ListShares 获取指定用户的分享列表（不包含已过期的分享）.
func (s *ShareService) ListShares(ctx context.Context, user string) (*types.ListSharesResponse, error) {
	if user == "" {
		return nil, fmt.Errorf("user is required")
	}

	if s.dbc == nil || s.dbc.GetDB() == nil {
		return nil, errors.New("db not initialized")
	}

	now := time.Now().UTC()

	var dbShares []model.Share
	if err := s.dbc.GetDB().
		Where("owner = ? AND (expire_at IS NULL OR expire_at > ?)", user, now).
		Order("created_at DESC").Find(&dbShares).Error; err != nil {
		return nil, fmt.Errorf("list shares: %w", err)
	}

	shares := make([]types.ShareInfo, 0, len(dbShares))
	for _, sh := range dbShares {
		rec, err := sh.ToRecord()
		if err != nil {
			return nil, err
		}

		shares = append(shares, (*shareRecord)(rec).toInfo())
	}

	return &types.ListSharesResponse{Shares: shares}, nil
}

// DeleteShare 删除指定的分享（仅 owner 可操作）.
func (s *ShareService) DeleteShare(ctx context.Context, user, shareID string) error {
	if user == "" || shareID == "" {
		return fmt.Errorf("user/shareID is required")
	}

	if s.dbc == nil || s.dbc.GetDB() == nil {
		return errors.New("db not initialized")
	}

	var sh model.Share
	if err := s.dbc.GetDB().Where("share_id = ?", shareID).First(&sh).Error; err != nil {
		return err
	}

	if sh.Owner != user {
		return fmt.Errorf("forbidden: not owner")
	}

	if err := s.dbc.GetDB().Delete(&sh).Error; err != nil {
		return err
	}
	// 删缓存
	_ = s.kvDel(ctx, makeShareKey(shareID))

	return nil
}

// GetShareDetail 获取分享详情（不包含密码等敏感信息）.
func (s *ShareService) GetShareDetail(ctx context.Context, shareID string) (*types.ShareInfo, error) {
	if shareID == "" {
		return nil, fmt.Errorf("shareID is required")
	}

	rec, err := s.getShareCached(ctx, shareID)
	if err != nil {
		return nil, err
	}

	info := rec.toInfo()

	return &info, nil
}

// AccessShare 访问分享，校验可选密码，返回分享信息（不包含密码等敏感信息）.
func (s *ShareService) AccessShare(ctx context.Context, shareID, password string) (*types.ShareInfo, error) {
	if shareID == "" {
		return nil, fmt.Errorf("shareID is required")
	}

	rec, err := s.getShareCached(ctx, shareID)
	if err != nil {
		return nil, err
	}

	if rec.PasswordHash != "" {
		if hashPassword(password) != rec.PasswordHash {
			return nil, fmt.Errorf("invalid password")
		}
	}

	info := rec.toInfo()

	return &info, nil
}

// GetShareDownloadURL 获取分享的下载直链（仅当允许下载且仅包含单个对象时有效）.
func (s *ShareService) GetShareDownloadURL(ctx context.Context, shareID string) (string, error) {
	if shareID == "" {
		return "", fmt.Errorf("shareID is required")
	}

	if s.s3c == nil {
		return "", errors.New("s3 not initialized")
	}

	rec, err := s.getShareCached(ctx, shareID)
	if err != nil {
		return "", err
	}

	if !rec.AllowDownload {
		return "", fmt.Errorf("download not allowed")
	}

	if len(rec.ObjectKeys) == 0 {
		return "", fmt.Errorf("no object in share")
	}

	if len(rec.ObjectKeys) > 1 {
		return "", fmt.Errorf("multiple objects not supported yet")
	}

	// 生成预签名下载（与 FileService 保持一致的默认过期时间）
	bucket, err := func() (string, error) {
		cfg := s.s3c.GetConfig()
		if len(cfg.Buckets) == 0 {
			return "", fmt.Errorf("no bucket configured")
		}

		return cfg.Buckets[0], nil
	}()
	if err != nil {
		return "", err
	}

	u, err := s.s3c.PresignedGetObject(ctx, bucket, rec.ObjectKeys[0], DefaultPresignedOpTimeout, nil)
	if err != nil {
		return "", fmt.Errorf("presign get: %w", err)
	}

	return u.String(), nil
}

// GetSharePermissions 获取分享权限.
func (s *ShareService) GetSharePermissions(ctx context.Context, user, shareID string) (*types.GetSharePermissionsResponse, error) {
	if user == "" || shareID == "" {
		return nil, fmt.Errorf("user/shareID is required")
	}

	rec, err := s.getShareCached(ctx, shareID)
	if err != nil {
		return nil, err
	}

	if rec.Owner != user {
		return nil, fmt.Errorf("forbidden: not owner")
	}

	return &types.GetSharePermissionsResponse{ShareID: shareID, Permissions: rec.Permissions}, nil
}

// UpdateSharePermissions 仅更新分享的权限信息（仅 owner 可操作）。
func (s *ShareService) UpdateSharePermissions(ctx context.Context, user, shareID string, req *types.UpdateSharePermissionsRequest) error {
	if user == "" || shareID == "" || req == nil {
		return fmt.Errorf("user/shareID/req is required")
	}

	rec, err := s.getShareCached(ctx, shareID)
	if err != nil {
		return err
	}

	if rec.Owner != user {
		return fmt.Errorf("forbidden: not owner")
	}

	rec.Permissions.AllowAnonymous = req.AllowAnonymous
	rec.Permissions.Users = slices.Compact(append([]string{}, req.Users...))

	if err := s.updatePermissionsInDB(ctx, shareID, rec.Permissions); err != nil {
		return err
	}
	// 刷新缓存
	_ = s.cacheShare(ctx, rec, ttlFromExpire(rec.ExpireAt))

	return nil
}

// AddShareUser 将 newUser 添加到 shareID 的访问用户列表中（仅限 owner 操作）.
func (s *ShareService) AddShareUser(ctx context.Context, user, shareID, newUser string) error {
	if user == "" || shareID == "" || newUser == "" {
		return fmt.Errorf("user/shareID/newUser is required")
	}

	rec, err := s.getShareCached(ctx, shareID)
	if err != nil {
		return err
	}

	if rec.Owner != user {
		return fmt.Errorf("forbidden: not owner")
	}

	if slices.Index(rec.Permissions.Users, newUser) < 0 {
		rec.Permissions.Users = append(rec.Permissions.Users, newUser)
	}

	if err := s.updatePermissionsInDB(ctx, shareID, rec.Permissions); err != nil {
		return err
	}

	_ = s.cacheShare(ctx, rec, ttlFromExpire(rec.ExpireAt))

	return nil
}

// RemoveShareUser 从 shareID 的访问用户列表中移除 target（仅限 owner 操作）.
// RemoveShareUser 将 target 从分享的访问用户列表中移除（仅 owner 可操作）。
func (s *ShareService) RemoveShareUser(ctx context.Context, user, shareID, target string) error {
	if user == "" || shareID == "" || target == "" {
		return fmt.Errorf("user/shareID/target is required")
	}

	rec, err := s.getShareCached(ctx, shareID)
	if err != nil {
		return err
	}

	if rec.Owner != user {
		return fmt.Errorf("forbidden: not owner")
	}

	out := rec.Permissions.Users[:0]
	for _, u := range rec.Permissions.Users {
		if u != target {
			out = append(out, u)
		}
	}

	rec.Permissions.Users = out

	if err := s.updatePermissionsInDB(ctx, shareID, rec.Permissions); err != nil {
		return err
	}

	_ = s.cacheShare(ctx, rec, ttlFromExpire(rec.ExpireAt))

	return nil
}

// ---- 内部模型与工具 ----

const (
	shareKeyPrefix = "shares:v1:"
)

// 缓存 TTL 策略常量：集中管理，避免魔数（mnd）。
const (
	shareCacheDefaultTTL = 10 * time.Minute // 未设置过期时间时的默认缓存时长
	shareCacheMaxTTL     = 30 * time.Minute // 单条分享缓存的最长缓存时间上限
)

// shareRecord 是 service 层使用的分享数据结构（与 model.ShareRecord 类似，但不依赖 model 包）.
type shareRecord struct {
	ShareID       string                 `json:"share_id"`
	Owner         string                 `json:"owner"`
	ObjectKeys    []string               `json:"object_keys"`
	CreatedAt     time.Time              `json:"created_at"`
	ExpireAt      *time.Time             `json:"expire_at,omitempty"`
	AllowDownload bool                   `json:"allow_download"`
	PasswordHash  string                 `json:"password_hash,omitempty"`
	Permissions   types.SharePermissions `json:"permissions"`
}

// toInfo 转换为对外的 ShareInfo 结构.
func (r *shareRecord) toInfo() types.ShareInfo {
	return types.ShareInfo{
		ShareID:       r.ShareID,
		Owner:         r.Owner,
		ObjectKeys:    r.ObjectKeys,
		CreatedAt:     r.CreatedAt,
		ExpireAt:      r.ExpireAt,
		AllowDownload: r.AllowDownload,
	}
}

func isExpired(now time.Time, exp *time.Time) bool {
	return exp != nil && now.After(*exp)
}

// newShareID 生成带前缀的 ULID 字符串，形如 "sh_01H..."。
// 使用单例熵源以支持同一毫秒内的单调递增。
func newShareID(t time.Time) string {
	// 注意：ULID 使用毫秒时间戳，因此应传入 time.Now().UTC() 或同等时间。
	id := ulid.MustNew(ulid.Timestamp(t), ulidEntropy)
	return "sh_" + id.String()
}

func makeShareKey(shareID string) string { return shareKeyPrefix + shareID }

func hashPassword(pw string) string {
	if strings.TrimSpace(pw) == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(pw))

	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// kvGet 通过 key 获取并反序列化值到 v，返回是否命中。
func (s *ShareService) kvGet(ctx context.Context, key string, v any) (bool, error) {
	if s.kvc == nil {
		return false, errors.New("kv client is nil")
	}

	b, err := s.kvc.Get(ctx, key)
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal(b, v); err != nil {
		return false, fmt.Errorf("unmarshal %s: %w", key, err)
	}

	return true, nil
}

// kvSet 序列化 v 并通过 key 存储，ttl 可选（0 表示不过期）。
func (s *ShareService) kvSet(ctx context.Context, key string, v any, ttl time.Duration) error {
	if s.kvc == nil {
		return errors.New("kv client is nil")
	}

	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", key, err)
	}

	return s.kvc.Set(ctx, key, b, ttl)
}

// kvDel 删除 key。
func (s *ShareService) kvDel(ctx context.Context, key string) error {
	if s.kvc == nil {
		return errors.New("kv client is nil")
	}

	return s.kvc.Delete(ctx, key)
}

// ---- DB 为主 + 轻缓存：转换/缓存与加载 ----

// fromModelRecord 将 DB 的 ShareRecord 转为 service 的 shareRecord。
func fromModelRecord(mr *model.ShareRecord) *shareRecord {
	if mr == nil {
		return nil
	}

	return &shareRecord{
		ShareID:       mr.ShareID,
		Owner:         mr.Owner,
		ObjectKeys:    append([]string{}, mr.ObjectKeys...),
		CreatedAt:     mr.CreatedAt,
		ExpireAt:      mr.ExpireAt,
		AllowDownload: mr.AllowDownload,
		PasswordHash:  mr.PasswordHash,
		Permissions:   mr.Permissions,
	}
}

// getShareCached 通过 shareID 获取 shareRecord 详细结构体，有限从缓存中读取，其次从 DB 回源.
func (s *ShareService) getShareCached(ctx context.Context, shareID string) (*shareRecord, error) {
	if shareID == "" {
		return nil, fmt.Errorf("shareID is required")
	}
	// 优先缓存
	if s.kvc != nil {
		var rec shareRecord
		if ok, err := s.kvGet(ctx, makeShareKey(shareID), &rec); err == nil && ok {
			if isExpired(time.Now().UTC(), rec.ExpireAt) {
				_ = s.kvDel(ctx, makeShareKey(shareID))
			} else {
				return &rec, nil
			}
		}
	}

	if s.dbc == nil || s.dbc.GetDB() == nil {
		return nil, errors.New("db not initialized")
	}
	// DB 加载
	var sh model.Share
	if err := s.dbc.GetDB().Where("share_id = ?", shareID).First(&sh).Error; err != nil {
		return nil, err
	}

	if isExpired(time.Now().UTC(), sh.ExpireAt) {
		return nil, fmt.Errorf("share expired")
	}

	recModel, err := sh.ToRecord()
	if err != nil {
		return nil, err
	}

	rec := fromModelRecord(recModel)
	// 回填缓存
	_ = s.cacheShare(ctx, rec, ttlFromExpire(rec.ExpireAt))

	return rec, nil
}

// cacheShare 将 rec 缓存到 KV，ttl 可选（0 表示不过期）.
func (s *ShareService) cacheShare(ctx context.Context, rec *shareRecord, ttl time.Duration) error {
	if s.kvc == nil || rec == nil {
		return nil
	}

	return s.kvSet(ctx, makeShareKey(rec.ShareID), rec, ttl)
}

// ttlFromExpire 根据过期时间计算缓存 TTL：
//   - 未设置过期：返回默认 TTL；
//   - 已设置过期：返回 [0, shareCacheMaxTTL] 范围内的值，避免长时间缓存导致权限更新不生效。
func ttlFromExpire(exp *time.Time) time.Duration {
	if exp == nil {
		return shareCacheDefaultTTL
	}

	d := time.Until(*exp)
	if d <= 0 {
		return 0
	}

	if d > shareCacheMaxTTL {
		return shareCacheMaxTTL
	}

	return d
}

// updatePermissionsInDB 仅更新 Share 的权限 JSON 与更新时间，避免覆盖其他字段。
func (s *ShareService) updatePermissionsInDB(_ context.Context, shareID string, perms types.SharePermissions) error {
	if s.dbc == nil || s.dbc.GetDB() == nil {
		return errors.New("db not initialized")
	}

	b, err := json.Marshal(perms)
	if err != nil {
		return err
	}

	return s.dbc.GetDB().Model(&model.Share{}).
		Where("share_id = ?", shareID).
		Updates(map[string]any{
			"permissions_json": string(b),
			"updated_at":       time.Now().UTC(),
		}).Error
}
