package tftype

type TFType int

const (
	TFUnknown TFType = iota
	TFBool    TFType = iota
	TFDynamic TFType = iota
	TFFloat64 TFType = iota
	TFInt64   TFType = iota
	TFNumber  TFType = iota
	TFObject  TFType = iota
	TFString  TFType = iota
)

func (t TFType) TypeName() string {
	switch t {
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
	}
	return ""
}
