package gobo

import (
	"reflect"
	"testing"
)

type NestedStruct struct {
	InnerField string `json:"inner_field" gobo:"A secret code"`
}

type TestResponse struct {
	ID        string         `json:"id" gobo:"A UUID v4"`
	Username  string         `json:"username" gobo:"A realistic internet handle"`
	Roles     []string       `json:"roles" gobo:"One of: admin, user, guest"`
	Nested    NestedStruct   `json:"nested"`
	MultiNest []NestedStruct `json:"multi_nest"`
	Ignored   string         // No json or gobo tag
}

func TestReflectSchema(t *testing.T) {
	fields := reflectSchema(TestResponse{})

	expectedFields := []FieldInfo{
		{Name: "ID", JSONName: "id", Type: "string", Interpreter: "A UUID v4"},
		{Name: "Username", JSONName: "username", Type: "string", Interpreter: "A realistic internet handle"},
		{Name: "Roles", JSONName: "roles", Type: "[]string", Interpreter: "One of: admin, user, guest"},
		{Name: "Nested", JSONName: "nested", Type: "gobo.NestedStruct", Interpreter: ""},
		{Name: "InnerField", JSONName: "nested.inner_field", Type: "string", Interpreter: "A secret code"},
		{Name: "MultiNest", JSONName: "multi_nest", Type: "[]gobo.NestedStruct", Interpreter: ""},
		{Name: "InnerField", JSONName: "multi_nest[*].inner_field", Type: "string", Interpreter: "A secret code"},
		{Name: "Ignored", JSONName: "Ignored", Type: "string", Interpreter: ""},
	}

	if len(fields) != len(expectedFields) {
		t.Fatalf("Expected %d fields, got %d", len(expectedFields), len(fields))
	}

	for i, expected := range expectedFields {
		actual := fields[i]
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Field %d mismatch.\nExpected: %+v\nGot:      %+v", i, expected, actual)
		}
	}
}

func TestFormatFieldInstructions(t *testing.T) {
	fields := []FieldInfo{
		{Name: "ID", JSONName: "id", Type: "string", Interpreter: "A UUID v4"},
		{Name: "Username", JSONName: "username", Type: "string", Interpreter: ""}, // no instruction
	}

	out := formatFieldInstructions(fields)
	expected := "- **`id`** (string): A UUID v4\n- **`username`** (string)\n"

	if out != expected {
		t.Errorf("Format output mismatch.\nExpected:\n%s\nGot:\n%s", expected, out)
	}
}
