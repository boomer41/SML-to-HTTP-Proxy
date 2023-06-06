package sml

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type fieldParams struct {
	optional                bool
	choiceHandler           string
	implicitChoiceAllowList []ImplicitChoiceHandler
}

type ChoiceHandler func(k string, keyToken smlToken) (interface{}, error)

func smlMessageChoiceHandler(k string, keyToken smlToken) (interface{}, error) {
	if k == "SML_MessageBody" {
		var valueId uint32
		foundValue := false

		// It SHOULD be an uint32, however, it MAY be encoded with fewer bits, when no ambiguity is created
		t, ok := keyToken.(*smlUnsigned32)

		if !ok {
			t2, ok := keyToken.(*smlUnsigned16)

			if !ok {
				t3, ok := keyToken.(*smlUnsigned8)

				if !ok {
					// fall through
				} else {
					valueId = uint32(t3.value)
					foundValue = true
				}
			} else {
				valueId = uint32(t2.value)
				foundValue = true
			}
		} else {
			valueId = t.value
			foundValue = true
		}

		if !foundValue {
			return nil, InvalidMessage{fmt.Errorf("expected uint32, got %v", keyToken)}
		}

		switch valueId {
		case 0x101:
			return &PublicOpenResMessageBody{}, nil
		case 0x201:
			return &PublicCloseResMessageBody{}, nil
		case 0x701:
			return &GetListResMessageBody{}, nil
		}

		return nil, InvalidMessage{fmt.Errorf("unsupported SML message %08x", valueId)}
	}

	return nil, fmt.Errorf("unsupported choice %s", k)
}

func deserializeMessageBundle(bundle *unparsedMessageBundle) (*File, error) {
	msgs := make([]*Message, 0)

	for _, bundleList := range bundle.messages {
		m := &Message{}

		err := deserializeField(reflect.ValueOf(m).Elem(), fieldParams{
			optional: false,
		}, bundleList, smlMessageChoiceHandler)

		if err != nil {
			if _, ok := err.(*InvalidMessage); !ok {
				return nil, err
			}

			return nil, &InvalidFile{
				error: err,
			}
		}

		msgs = append(msgs, m)
	}

	if len(msgs) < 2 {
		return nil, &InvalidFile{
			errors.New("SML file must contain at least two messages"),
		}
	}

	if _, ok := msgs[0].MessageBody.(*PublicOpenResMessageBody); !ok {
		return nil, &InvalidFile{
			errors.New("SML file must begin with a SML_PublicOpen.Res message"),
		}
	}

	if _, ok := msgs[len(msgs)-1].MessageBody.(*PublicCloseResMessageBody); !ok {
		return nil, &InvalidFile{
			errors.New("SML file must end with a SML_PublicClose.Res message"),
		}
	}

	for _, m := range msgs[1 : len(msgs)-1] {
		_, isPublicOpen := m.MessageBody.(*PublicOpenResMessageBody)
		_, isPublicClose := m.MessageBody.(*PublicCloseResMessageBody)

		if isPublicOpen || isPublicClose {
			return nil, &InvalidFile{
				errors.New("SML file must not contain a SML_PublicOpen.Res or SML_PublicClose.Res message in the middle of the file"),
			}
		}
	}

	return &File{
		Messages: msgs,
	}, nil
}

func deserializeField(v reflect.Value, params fieldParams, token smlToken, choiceHandler ChoiceHandler) error {
	var err error

	switch v.Kind() {
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			bytes, err := deserializeOctetString(token)

			if err != nil {
				return err
			}

			if len(bytes) != 0 || !params.optional {
				v.Set(reflect.MakeSlice(v.Type(), len(bytes), len(bytes)))
				reflect.Copy(v, reflect.ValueOf(bytes))
			}

			return nil
		} else if (v.Type().Elem().Kind() == reflect.Pointer && v.Type().Elem().Elem().Kind() == reflect.Struct) || v.Type().Elem().Kind() == reflect.Interface {
			// In any case, we need list
			list, ok := token.(*smlList)

			if !ok {
				return InvalidMessage{fmt.Errorf("deserializing a slice of pointers to structs or interfaces requires a list")}
			}

			slice := reflect.MakeSlice(reflect.SliceOf(v.Type().Elem()), 0, len(list.value))

			for _, listElement := range list.value {
				if v.Type().Elem().Kind() == reflect.Interface {
					var value interface{}

					err = deserializeField(reflect.ValueOf(value), params, listElement, choiceHandler)

					if err != nil {
						return err
					}

					slice = reflect.Append(slice, reflect.ValueOf(value))
				} else {
					value := reflect.New(v.Type().Elem().Elem())
					err = deserializeField(value.Elem(), params, listElement, choiceHandler)

					if err != nil {
						return err
					}

					slice = reflect.Append(slice, value)
				}
			}

			v.Set(slice)
			return nil
		} else {
			return fmt.Errorf("unsupported slice element type %v", v.Type().Elem().Kind())
		}
	case reflect.Struct:
		for i := 0; i < v.Type().NumField(); i++ {
			if !v.Type().Field(i).IsExported() {
				return errors.New("struct contains unexported fields")
			}
		}

		list, ok := token.(*smlList)

		if !ok {
			if _, ok := token.(*smlOctetString); params.optional && ok {
				return nil
			}

			return &InvalidMessage{errors.New("struct needs to be decoded upon a list")}
		}

		if len(list.value) != v.Type().NumField() {
			return &InvalidMessage{errors.New("struct size mismatch against decoded data")}
		}

		for i := 0; i < v.Type().NumField(); i++ {
			fieldName := v.Type().Field(i).Name
			_ = fieldName

			p, err := parseFieldParams(v.Type().Field(i))

			if err != nil {
				return err
			}

			err = deserializeField(v.Field(i), p, list.value[i], choiceHandler)

			if err != nil {
				return err
			}
		}

		return nil

	// Choice
	case reflect.Interface:
		if params.implicitChoiceAllowList != nil {
			return decodeImplicitChoice(v, params, token)
		}

		if params.choiceHandler == "" {
			v.Set(reflect.ValueOf(token))
			return nil
		}

		choiceList, ok := token.(*smlList)

		if !ok || len(choiceList.value) != 2 {
			if _, ok := token.(*smlOctetString); params.optional && ok {
				return nil
			}

			return InvalidMessage{errors.New("choice must be deserialized using a list with 2 elements")}
		}

		interfaceValue, err := choiceHandler(params.choiceHandler, choiceList.value[0])

		if err != nil {
			return err
		}

		interfaceValueReflect := reflect.ValueOf(interfaceValue)

		if interfaceValueReflect.Kind() != reflect.Pointer || interfaceValueReflect.Elem().Kind() != reflect.Struct {
			return errors.New("choice handler must return a pointer to a struct")
		}

		v.Set(interfaceValueReflect)

		return deserializeField(v.Elem().Elem(), params, choiceList.value[1], choiceHandler)
	case reflect.Uint8:
		tok, ok := token.(*smlUnsigned8)

		if !ok {
			if octetString, ok := token.(*smlOctetString); ok && params.optional && len(octetString.value) == 0 {
				tok = &smlUnsigned8{
					value: 0,
				}
			} else {
				return &InvalidMessage{fmt.Errorf("type mismatch. expected %s, got %v", "uint8", token)}
			}
		}

		v.SetUint(uint64(tok.value))
		return nil
	case reflect.Uint16:
		tok, ok := token.(*smlUnsigned16)

		if !ok {
			if octetString, ok := token.(*smlOctetString); ok && params.optional && len(octetString.value) == 0 {
				tok = &smlUnsigned16{
					value: 0,
				}
			} else {
				return &InvalidMessage{fmt.Errorf("type mismatch. expected %s, got %v", "uint16", token)}
			}
		}

		v.SetUint(uint64(tok.value))
		return nil
	case reflect.Uint32:
		tok, ok := token.(*smlUnsigned32)

		if !ok {
			if octetString, ok := token.(*smlOctetString); ok && params.optional && len(octetString.value) == 0 {
				tok = &smlUnsigned32{
					value: 0,
				}
			} else {
				return &InvalidMessage{fmt.Errorf("type mismatch. expected %s, got %v", "uint32", token)}
			}
		}

		v.SetUint(uint64(tok.value))
		return nil
	case reflect.Uint64:
		tok, ok := token.(*smlUnsigned64)

		if !ok {
			if octetString, ok := token.(*smlOctetString); ok && params.optional && len(octetString.value) == 0 {
				tok = &smlUnsigned64{
					value: 0,
				}
			} else {
				return &InvalidMessage{fmt.Errorf("type mismatch. expected %s, got %v", "uint64", token)}
			}
		}

		v.SetUint(tok.value)
		return nil
	case reflect.Int8:
		tok, ok := token.(*smlSigned8)

		if !ok {
			if octetString, ok := token.(*smlOctetString); ok && params.optional && len(octetString.value) == 0 {
				tok = &smlSigned8{
					value: 0,
				}
			} else {
				return &InvalidMessage{fmt.Errorf("type mismatch. expected %s, got %v", "int8", token)}
			}
		}

		v.SetInt(int64(tok.value))
		return nil
	case reflect.Int16:
		tok, ok := token.(*smlSigned16)

		if !ok {
			if octetString, ok := token.(*smlOctetString); ok && params.optional && len(octetString.value) == 0 {
				tok = &smlSigned16{
					value: 0,
				}
			} else {
				return &InvalidMessage{fmt.Errorf("type mismatch. expected %s, got %v", "int16", token)}
			}
		}

		v.SetInt(int64(tok.value))
		return nil
	case reflect.Int32:
		tok, ok := token.(*smlSigned32)

		if !ok {
			if octetString, ok := token.(*smlOctetString); ok && params.optional && len(octetString.value) == 0 {
				tok = &smlSigned32{
					value: 0,
				}
			} else {
				return &InvalidMessage{fmt.Errorf("type mismatch. expected %s, got %v", "int32", token)}
			}
		}

		v.SetInt(int64(tok.value))
		return nil
	case reflect.Int64:
		tok, ok := token.(*smlSigned64)

		if !ok {
			if octetString, ok := token.(*smlOctetString); ok && params.optional && len(octetString.value) == 0 {
				tok = &smlSigned64{
					value: 0,
				}
			} else {
				return &InvalidMessage{fmt.Errorf("type mismatch. expected %s, got %v", "int64", token)}
			}
		}

		v.SetInt(tok.value)
		return nil
	default:
		return fmt.Errorf("unsupported reflection type %v", v.Kind())
	}
}

func parseFieldParams(v reflect.StructField) (fieldParams, error) {
	tag, ok := v.Tag.Lookup("sml")

	if !ok {
		return fieldParams{}, nil
	}

	tagSplit := strings.Split(tag, ",")
	p := fieldParams{}

	for _, v := range tagSplit {
		kvSplit := strings.Split(v, ":")

		switch kvSplit[0] {
		case "optional":
			p.optional = true
		case "choice":
			if len(kvSplit) != 2 {
				return fieldParams{}, errors.New("choice tag requires value")
			}

			p.choiceHandler = kvSplit[1]
		case "implicit_choice":
			list, err := parseImplicitChoiceHandlers(kvSplit[1:])

			if err != nil {
				return fieldParams{}, err
			}

			p.implicitChoiceAllowList = list
		default:
			return fieldParams{}, fmt.Errorf("unkown tag value %s", v)
		}
	}

	return p, nil
}

func deserializeOctetString(token smlToken) ([]byte, error) {
	octetString, ok := token.(*smlOctetString)

	if !ok {
		return nil, &InvalidMessage{fmt.Errorf("expected octet string, got %v", token)}
	}

	return octetString.value, nil
}
