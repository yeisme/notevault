package kv

import (
	"bytes"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
)

// 跨实现的KV值的常见TTL包装器.
const ttlMagic = "NVTTL1:"

type ttlValue struct {
	V []byte `json:"v"`
	E int64  `json:"e,omitempty"` // unix seconds; 0 means no expiry
}

// encodeWithTTL wraps the value when ttl>0; otherwise returns original value.
func encodeWithTTL(value []byte, ttl time.Duration) ([]byte, bool, error) {
	if ttl <= 0 {
		return value, false, nil
	}

	tv := ttlValue{V: value}
	tv.E = time.Now().Add(ttl).Unix()

	b, err := sonic.Marshal(tv)
	if err != nil {
		return nil, false, fmt.Errorf("marshal ttl value: %w", err)
	}

	out := append([]byte(ttlMagic), b...)

	return out, true, nil
}

// decodeWithTTL detects wrapper and decides expiration status.
// Returns (value, expired, wrapped, error).
func decodeWithTTL(b []byte, now time.Time) ([]byte, bool, bool, error) {
	if !bytes.HasPrefix(b, []byte(ttlMagic)) {
		return b, false, false, nil
	}

	var tv ttlValue
	if err := sonic.Unmarshal(b[len(ttlMagic):], &tv); err != nil {
		return nil, false, true, fmt.Errorf("unmarshal ttl value: %w", err)
	}

	if tv.E > 0 && now.Unix() >= tv.E {
		return nil, true, true, nil
	}

	return tv.V, false, true, nil
}
