package gobo

import (
	"fmt"
	"reflect"
	"strings"
)

// FieldInfo contains information about a reflected struct field.
type FieldInfo struct {
	Name        string
	JSONName    string
	Type        string
	Interpreter string // The custom "gobo" tag instruction
}

// reflectSchema recursively visits a struct and extracts information about its fields, including custom "gobo" tags.
func reflectSchema(v any) []FieldInfo {
	return reflectValue(reflect.ValueOf(v), "")
}

// reflectValue recursively traverses a reflect.Value and collects FieldInfo.
func reflectValue(val reflect.Value, prefix string) []FieldInfo {
	var fields []FieldInfo

	// Unpack interfaces and pointers
	if val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr {
		if val.IsNil() {
			// If it's a nil pointer/interface but we know the type, we can create a zero value
			// to reflect its fields.
			val = reflect.Zero(val.Type().Elem())
		} else {
			val = val.Elem()
		}
	}

	switch val.Kind() {
	case reflect.Struct:
		typ := val.Type()
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)

			// Ignore unexported fields
			if !field.IsExported() {
				continue
			}

			jsonTag := field.Tag.Get("json")
			jsonName := field.Name
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" {
					jsonName = parts[0]
				}
			}

			// Extract our custom tag
			goboTag := field.Tag.Get("gobo")

			fullName := jsonName
			if prefix != "" {
				fullName = prefix + "." + jsonName
			}

			// Add the current field
			fields = append(fields, FieldInfo{
				Name:        field.Name,
				JSONName:    fullName,
				Type:        field.Type.String(),
				Interpreter: goboTag,
			})

			// Recurse into nested structs or slices of structs
			fieldVal := val.Field(i)
			if field.Type.Kind() == reflect.Struct {
				fields = append(fields, reflectValue(fieldVal, fullName)...)
			} else if field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Array {
				elemType := field.Type.Elem()
				// If it's a slice of pointers, get the underlying struct type
				if elemType.Kind() == reflect.Ptr {
					elemType = elemType.Elem()
				}
				if elemType.Kind() == reflect.Struct {
					// create a zero value to reflect
					elemVal := reflect.Zero(elemType)
					// Use [*] to indicate it's an array element in the JSON path
					fields = append(fields, reflectValue(elemVal, fullName+"[*]")...)
				}
			}
		}
	case reflect.Slice, reflect.Array:
		elemType := val.Type().Elem()
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
		if elemType.Kind() == reflect.Struct {
			elemVal := reflect.Zero(elemType)
			fields = append(fields, reflectValue(elemVal, prefix+"[*]")...)
		}
	}

	return fields
}

// formatFieldInstructions formats the extracted field information into a markdown-like list for the LLM.
func formatFieldInstructions(fields []FieldInfo) string {
	if len(fields) == 0 {
		return "No specific field instructions found."
	}

	var sb strings.Builder
	for _, f := range fields {
		sb.WriteString(fmt.Sprintf("- **`%s`** (%s)", f.JSONName, f.Type))
		if f.Interpreter != "" {
			sb.WriteString(fmt.Sprintf(": %s", f.Interpreter))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
