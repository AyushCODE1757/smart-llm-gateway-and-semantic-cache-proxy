package cache

import (
	"crypto/sha256"
	"encoding/hex"
)

const cacheKeyPrefix = "llm:cache:"

func GenerateCacheKey(input string) string {
	sum := sha256.Sum256([]byte(input))
	return cacheKeyPrefix + hex.EncodeToString(sum[:])
}
