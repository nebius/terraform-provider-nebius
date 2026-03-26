package validators

import (
	"math"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func Int32Validator() validator.Int64 {
	return int64validator.Between(math.MinInt32, math.MaxInt32)
}

func MapInt32Validator() validator.Map {
	return mapvalidator.ValueInt64sAre(Int32Validator())
}

func ListInt32Validator() validator.List {
	return listvalidator.ValueInt64sAre(Int32Validator())
}
