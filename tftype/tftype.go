package tftype

type TFType int

const (
	TFUnknown TFType = iota
	TFBool
	TFDynamic
	TFFloat64
	TFInt64
	TFNumber
	TFObject
	TFString
)

func (t TFType) TypeName() string {
	switch t {
	case TFUnknown:
		return ""
	case TFBool:
		return "Bool"
	case TFDynamic:
		return "Dynamic"
	case TFFloat64:
		return "Float64"
	case TFInt64:
		return "Int64"
	case TFNumber:
		return "Number"
	case TFObject:
		return "Object"
	case TFString:
		return "String"
	default:
		return ""
	}
}
