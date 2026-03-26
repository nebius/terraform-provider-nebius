package normalizers

import (
	"bytes"
	"encoding/pem"
	"fmt"
	"strings"
)

func init() {
	Register(NewNormalizerError(
		"pem_blocks",
		"Parses one or more PEM-encoded blocks and returns their canonical PEM representation.",
		pemBlocksNormalize,
	))
}

func pemBlocksNormalize(input string) (string, error) {
	content := bytes.TrimSpace([]byte(input))
	if len(content) == 0 {
		return "", nil
	}

	var blocks []*pem.Block
	for len(content) > 0 {
		block, rest := pem.Decode(content)
		if block == nil {
			return "", fmt.Errorf("failed to decode PEM block")
		}
		blocks = append(blocks, block)
		content = bytes.TrimSpace(rest)
	}

	normalized := strings.Builder{}
	for _, block := range blocks {
		if err := pem.Encode(&normalized, block); err != nil {
			return "", fmt.Errorf("encode pem block: %w", err)
		}
	}

	return normalized.String(), nil
}
