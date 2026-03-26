package normalizers

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type compositionNormalizer struct {
	name        string
	normalizers []Normalizer
}

func (n *compositionNormalizer) Normalize(input string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	normalizedValue := input

	for _, normalizer := range n.normalizers {
		var normDiags diag.Diagnostics
		normalizedValue, normDiags = normalizer.Normalize(normalizedValue)
		diags.Append(normDiags...)
		if diags.HasError() {
			return "", diags
		}
	}

	return normalizedValue, diags
}

func (n *compositionNormalizer) Documentation() string {
	docs := "Applies the following normalization rules in order:\n"
	for _, normalizer := range n.normalizers {
		docs += "- " + normalizer.Name() + ": " + normalizer.Documentation() + "\n"
	}
	return docs
}

func (n *compositionNormalizer) Name() string {
	return n.name
}

func NewCompositionNormalizer(name string, normalizers ...string) (Normalizer, error) {
	normalizedList := make([]Normalizer, 0, len(normalizers))
	for _, normalizerName := range normalizers {
		normalizer := Get(normalizerName)
		if normalizer == nil {
			return nil, fmt.Errorf("unknown normalizer name in composition: %s", normalizerName)
		}
		normalizedList = append(normalizedList, normalizer)
	}
	return &compositionNormalizer{
		name:        name,
		normalizers: normalizedList,
	}, nil
}
func NewCompositionNormalizerMust(name string, normalizers ...string) Normalizer {
	normalizer, err := NewCompositionNormalizer(name, normalizers...)
	if err != nil {
		panic(err)
	}
	return normalizer
}
