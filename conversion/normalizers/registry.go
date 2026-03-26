package normalizers

import (
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type Normalizer interface {
	Normalize(input string) (string, diag.Diagnostics)
	Documentation() string
	Name() string
}

var (
	normalizersRegistryMap = map[string]Normalizer{}
	normalizersMu          sync.RWMutex
)

func Get(name string) Normalizer {
	normalizersMu.RLock()
	normalizer, exists := normalizersRegistryMap[name]
	normalizersMu.RUnlock()
	if !exists {
		return nil
	}
	return normalizer
}

func Register(normalizer ...Normalizer) {
	normalizersMu.Lock()
	for _, n := range normalizer {
		if _, exists := normalizersRegistryMap[n.Name()]; exists {
			normalizersMu.Unlock()
			panic("duplicate normalizer name in registry: " + n.Name())
		}
		normalizersRegistryMap[n.Name()] = n
	}
	normalizersMu.Unlock()
}

func init() {
	Register(NewNormalizerSimple(
		"lowercase",
		"Converts all characters in the string to lowercase.",
		strings.ToLower,
	))
	Register(NewNormalizerSimple(
		"trim_left",
		"Removes all leading whitespace characters (spaces, tabs, newlines, etc.) from the string.",
		func(input string) string { return strings.TrimLeft(input, " \t\n\r") },
	))
	Register(NewNormalizerSimple(
		"trim_right",
		"Removes all trailing whitespace characters (spaces, tabs, newlines, etc.) from the string.",
		func(input string) string { return strings.TrimRight(input, " \t\n\r") },
	))
	Register(NewCompositionNormalizerMust("trim", "trim_left", "trim_right"))
}
