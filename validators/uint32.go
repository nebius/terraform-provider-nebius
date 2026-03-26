package validators

import (
	"math"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func Uint32Validator() validator.Int64 {
	return int64validator.Between(0, math.MaxUint32)
}

func MapUint32Validator() validator.Map {
	return mapvalidator.ValueInt64sAre(Uint32Validator())
}

func ListUint32Validator() validator.List {
	return listvalidator.ValueInt64sAre(Uint32Validator())
}
