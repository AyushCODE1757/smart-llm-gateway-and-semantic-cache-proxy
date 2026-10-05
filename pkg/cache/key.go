package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"llm-gateway/pkg/types"
	"strings"
)

const cacheKeyPrefix = "llm:cache:"

func GenerateCacheKey(input string) string {
	sum := sha256.Sum256([]byte(input))
	return cacheKeyPrefix + hex.EncodeToString(sum[:])
}

func BuildCachePrompt(req types.ChatCompletionRequest) string {
	var builder strings.Builder
	builder.WriteString("model:")
	builder.WriteString(req.Model)
	builder.WriteString("\n")
	builder.WriteString("temperature:")
	builder.WriteString(fmt.Sprintf("%.6f", req.Temperature))
	builder.WriteString("\n")
	for _, message := range req.Messages {
		builder.WriteString(message.Role)
		builder.WriteString(":")
		builder.WriteString(message.Content)
		builder.WriteString("\n")
	}

	return builder.String()

}
