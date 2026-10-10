// Package mapper holds struct-to-struct and struct-to-map helpers.
package mapper

import (
	"encoding/json/v2"
	"reflect"

	"github.com/jinzhu/copier"
)

// Copy - copy struct to struct
func Copy(dest, src any) error {
	return copier.Copy(dest, src)
}

// CopyIgnoreEmpty - copy struct to struct ignore zero value
func CopyIgnoreEmpty(dest, src any) error {
	return copier.CopyWithOption(dest, src, copier.Option{IgnoreEmpty: true})
}

// BindingStruct - biding struct to struct
func BindingStruct(src any, desc any) error {
	byteSrc, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(byteSrc, desc)
}

// BindingAndValidate marshals detail into a T and runs validator over the result.
func BindingAndValidate[T any](detail any, validator func(any) error) (T, error) {
	var model T
	if err := BindingStruct(detail, &model); err != nil {
		return model, err
	}

	if err := validator(model); err != nil {
		return model, err
	}
	return model, nil
}

// StructToMap flattens a struct into a map keyed by each field's json tag. Fields without a json tag are skipped; nil pointer
// fields are dropped when ignoreNilField is true and mapped to nil otherwise.
func StructToMap(input any, ignoreNilField bool) map[string]any {
	result := make(map[string]any)
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("json") == "" {
			continue
		}

		fv := v.Field(i)
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				if ignoreNilField {
					continue
				}
				result[field.Tag.Get("json")] = nil
				continue
			}
			fv = fv.Elem()
		}

		result[field.Tag.Get("json")] = fv.Interface()
	}
	return result
}

// GetJSONStringify converts a struct to a JSON string, excluding specified fields.
func GetJSONStringify(src any) string {
	byteData, err := json.Marshal(src)
	if err != nil {
		return ""
	}
	return string(byteData)
}

// ConvertMapToString renders data as a JSON string, or "" when it cannot be marshalled.
func ConvertMapToString(data map[string]any) string {
	byteData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(byteData)
}
