package asr

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"voice-assistant/app/config"

	nls "github.com/aliyun/alibabacloud-nls-go-sdk"
)

// tokenRefreshBuffer 提前刷新缓冲：在 token 到期前 1 小时即视为过期，避免临界过期握手失败。
const tokenRefreshBuffer = time.Hour

// tokenKey 标识一份 ConnectionConfig：url+appkey+akid 共同决定一个缓存条目。
type tokenKey struct {
	url    string
	appkey string
	akid   string
}

// tokenManager 是线程安全的 NLS token 缓存单例。
type tokenManager struct {
	mu      sync.Mutex
	entries map[tokenKey]*tokenEntry
}

type tokenEntry struct {
	token      string
	expireAt   time.Time
	connConfig *nls.ConnectionConfig
}

var tokenMgr = &tokenManager{entries: make(map[tokenKey]*tokenEntry)}

// GetConnectionConfig 返回带有效 token 的 ConnectionConfig。
// 命中未过期缓存时直接复用，否则通过 AccessKey 调用 CreateToken 刷新。
func GetConnectionConfig(url, appkey, akid, akkey string) (*nls.ConnectionConfig, error) {
	if akid == "" || akkey == "" {
		return nil, errors.New("NLS AccessKeyId/AccessKeySecret is required (config.asr.accessKeyId / config.tts.accessKeyId)")
	}

	key := tokenKey{url: url, appkey: appkey, akid: akid}

	tokenMgr.mu.Lock()
	defer tokenMgr.mu.Unlock()

	// 命中且未过期（含 1h 缓冲）：直接复用
	if entry, ok := tokenMgr.entries[key]; ok && entry.token != "" {
		if time.Now().Before(entry.expireAt.Add(-tokenRefreshBuffer)) {
			return entry.connConfig, nil
		}
	}

	// 重新获取 token
	tokenMsg, err := nls.GetToken(nls.DEFAULT_DISTRIBUTE, nls.DEFAULT_DOMAIN, akid, akkey, nls.DEFAULT_VERSION)
	if err != nil {
		return nil, fmt.Errorf("NLS GetToken failed: %w", err)
	}
	if tokenMsg.TokenResult.Id == "" {
		return nil, fmt.Errorf("NLS GetToken returned empty token: %s", tokenMsg.ErrMsg)
	}

	connCfg := nls.NewConnectionConfigWithToken(url, appkey, tokenMsg.TokenResult.Id)
	expireAt := time.Unix(tokenMsg.TokenResult.ExpireTime, 0)
	if expireAt.IsZero() || expireAt.Before(time.Now()) {
		// SDK 极少返回 0，兜底按 24h 处理
		expireAt = time.Now().Add(24 * time.Hour)
	}

	tokenMgr.entries[key] = &tokenEntry{
		token:      tokenMsg.TokenResult.Id,
		expireAt:   expireAt,
		connConfig: connCfg,
	}
	return connCfg, nil
}

// GetConnectionConfigForASR 从 config.Config.Asr 读取 AccessKey 构造 ConnectionConfig。
func GetConnectionConfigForASR() (*nls.ConnectionConfig, error) {
	cfg := config.Config.Asr
	return GetConnectionConfig(nls.DEFAULT_URL, cfg.AppKey, cfg.AccessKeyId, cfg.AccessKeySecret)
}

// GetConnectionConfigForTTS 从 config.Config.Tts 读取 AccessKey 构造 ConnectionConfig。
func GetConnectionConfigForTTS() (*nls.ConnectionConfig, error) {
	cfg := config.Config.Tts
	return GetConnectionConfig(nls.DEFAULT_URL, cfg.AppKey, cfg.AccessKeyId, cfg.AccessKeySecret)
}
