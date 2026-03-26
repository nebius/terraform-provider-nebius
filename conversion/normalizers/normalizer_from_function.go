package normalizers

import "github.com/hashicorp/terraform-plugin-framework/diag"

var (
	_ Normalizer = (*normalizerFromFunc)(nil)
)

type normalizerFromFunc struct {
	normalizeFunc    func(input string) (string, diag.Diagnostics)
	documentationStr string
	nameStr          string
}

func (n *normalizerFromFunc) Normalize(input string) (string, diag.Diagnostics) {
	return n.normalizeFunc(input)
}

func (n *normalizerFromFunc) Documentation() string {
	return n.documentationStr
}

func (n *normalizerFromFunc) Name() string {
	return n.nameStr
}

func NewNormalizerDiagnostics(
	name string,
	documentation string,
	normalizeFunc func(input string) (string, diag.Diagnostics),
) Normalizer {
	return &normalizerFromFunc{
		normalizeFunc:    normalizeFunc,
		documentationStr: documentation,
		nameStr:          name,
	}
}

func NewNormalizerSimple(
	name string,
	documentation string,
	normalizeFunc func(input string) string,
) Normalizer {
	return &normalizerFromFunc{
		normalizeFunc: func(input string) (string, diag.Diagnostics) {
			return normalizeFunc(input), nil
		},
		documentationStr: documentation,
		nameStr:          name,
	}
}

func NewNormalizerError(
	name string,
	documentation string,
	normalizeFunc func(input string) (string, error),
) Normalizer {
	return &normalizerFromFunc{
		normalizeFunc: func(input string) (string, diag.Diagnostics) {
			var diags diag.Diagnostics
			result, err := normalizeFunc(input)
			if err != nil {
				diags.AddError(
					"Normalization Error: "+name,
					"An error occurred during normalization: "+err.Error(),
				)
			}
			return result, diags
		},
		documentationStr: documentation,
		nameStr:          name,
	}
}
