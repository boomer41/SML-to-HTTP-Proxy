package sml

import (
	"errors"
	"fmt"
	"reflect"
)

type ImplicitChoiceHandler func(v reflect.Value, token smlToken) (bool, error)

func parseImplicitChoiceHandlers(values []string) ([]ImplicitChoiceHandler, error) {
	h := make([]ImplicitChoiceHandler, 0)

	for _, v := range values {
		switch v {
		case "bool":
			h = append(h, decodeImplicitChoiceBoolean)
		case "octet_string":
			h = append(h, decodeImplicitChoiceOctetString)
		case "uint8":
			h = append(h, decodeImplicitChoiceUint8)
		case "uint16":
			h = append(h, decodeImplicitChoiceUint16)
		case "uint32":
			h = append(h, decodeImplicitChoiceUint32)
		case "uint64":
			h = append(h, decodeImplicitChoiceUint64)
		case "int8":
			h = append(h, decodeImplicitChoiceInt8)
		case "int16":
			h = append(h, decodeImplicitChoiceInt16)
		case "int32":
			h = append(h, decodeImplicitChoiceInt32)
		case "int64":
			h = append(h, decodeImplicitChoiceInt64)
		default:
			return nil, fmt.Errorf("unsupported implicit choice type %v", v)
		}
	}

	return h, nil
}

func decodeImplicitChoice(v reflect.Value, params fieldParams, token smlToken) error {
	for _, handler := range params.implicitChoiceAllowList {
		success, err := handler(v, token)

		if err != nil {
			return err
		}

		if success {
			return nil
		}
	}

	if params.optional {
		octetString, ok := token.(*smlOctetString)

		if ok && len(octetString.value) == 0 {
			return nil
		}
	}

	return errors.New("no implicit choice handler matched")
}

func decodeImplicitChoiceBoolean(v reflect.Value, token smlToken) (bool, error) {
	t, ok := token.(*smlBoolean)

	if !ok {
		return false, nil
	}

	v.SetBool(t.value)
	return true, nil
}

func decodeImplicitChoiceOctetString(v reflect.Value, token smlToken) (bool, error) {
	t, ok := token.(*smlOctetString)

	if !ok {
		return false, nil
	}

	slice := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(uint8(0))), len(t.value), len(t.value))
	reflect.Copy(slice, reflect.ValueOf(t.value))
	v.Set(slice)
	return true, nil
}

func decodeImplicitChoiceUint8(v reflect.Value, token smlToken) (bool, error) {
	t, ok := token.(*smlUnsigned8)

	if !ok {
		return false, nil
	}

	val := reflect.New(reflect.TypeOf(t.value))
	val.Elem().SetUint(uint64(t.value))
	v.Set(val)
	return true, nil
}

func decodeImplicitChoiceUint16(v reflect.Value, token smlToken) (bool, error) {
	t, ok := token.(*smlUnsigned16)

	if !ok {
		return false, nil
	}

	val := reflect.New(reflect.TypeOf(t.value))
	val.Elem().SetUint(uint64(t.value))
	v.Set(val)
	return true, nil
}

func decodeImplicitChoiceUint32(v reflect.Value, token smlToken) (bool, error) {
	t, ok := token.(*smlUnsigned32)

	if !ok {
		return false, nil
	}

	val := reflect.New(reflect.TypeOf(t.value))
	val.Elem().SetUint(uint64(t.value))
	v.Set(val)
	return true, nil
}

func decodeImplicitChoiceUint64(v reflect.Value, token smlToken) (bool, error) {
	t, ok := token.(*smlUnsigned64)

	if !ok {
		return false, nil
	}

	val := reflect.New(reflect.TypeOf(t.value))
	val.Elem().SetUint(t.value)
	v.Set(val)
	return true, nil
}

func decodeImplicitChoiceInt8(v reflect.Value, token smlToken) (bool, error) {
	t, ok := token.(*smlSigned8)

	if !ok {
		return false, nil
	}

	val := reflect.New(reflect.TypeOf(t.value))
	val.Elem().SetInt(int64(t.value))
	v.Set(val)
	return true, nil
}

func decodeImplicitChoiceInt16(v reflect.Value, token smlToken) (bool, error) {
	t, ok := token.(*smlSigned16)

	if !ok {
		return false, nil
	}

	val := reflect.New(reflect.TypeOf(t.value))
	val.Elem().SetInt(int64(t.value))
	v.Set(val)
	return true, nil
}

func decodeImplicitChoiceInt32(v reflect.Value, token smlToken) (bool, error) {
	t, ok := token.(*smlSigned32)

	if !ok {
		return false, nil
	}

	val := reflect.New(reflect.TypeOf(t.value))
	val.Elem().SetInt(int64(t.value))
	v.Set(val)
	return true, nil
}

func decodeImplicitChoiceInt64(v reflect.Value, token smlToken) (bool, error) {
	t, ok := token.(*smlSigned64)

	if !ok {
		return false, nil
	}

	val := reflect.New(reflect.TypeOf(t.value))
	val.Elem().SetInt(int64(t.value))
	v.Set(val)
	return true, nil
}
