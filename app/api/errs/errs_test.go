package errs_test

import (
	"reflect"
	"testing"

	"github.com/hamidoujand/task-scheduler/app/api/errs"
)

func TestAppValidator_Check(t *testing.T) {
	appValidator, err := errs.NewAppValidator()
	if err != nil {
		t.Fatalf("should be able to construct an app validator with default translator set to english: %s", err)
	}

	type Data struct {
		input  any
		fields map[string]string
		check  bool
	}

	tests := map[string]Data{
		"pass validation": {
			input: struct {
				Name string `json:"name" validate:"required"`
				Age  int    `json:"age" validate:"required,gte=18"`
			}{
				Name: "John",
				Age:  19,
			},
			fields: nil,
			check:  true,
		},

		"fail validation": {
			input: struct {
				Name string `json:"name" validate:"required"`
				Age  int    `json:"age" validate:"required,gte=18"`
			}{},
			fields: map[string]string{
				"age":  "age is a required field",
				"name": "name is a required field",
			},
			check: false,
		},
	}

	for k, v := range tests {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			fields, isOk := appValidator.Check(v.input)
			if v.check != isOk {
				//failed
				t.Errorf("expected to pass, but failed %t", isOk)
			}
			if !reflect.DeepEqual(fields, v.fields) {
				t.Errorf("expected the fields map to be equal with result")
			}
		})
	}
}

func TestCustomValidators(t *testing.T) {
	data := struct {
		Command string   `json:"command" validate:"required,commonCommands"`
		Args    []string `json:"args" validate:"required,commonArgs"`
	}{
		Command: "rm",
		Args:    []string{";"},
	}
	appValidator, err := errs.NewAppValidator()
	if err != nil {
		t.Fatalf("should be able to construct an app validator with default translator set to english: %s", err)
	}

	fields, ok := appValidator.Check(data)

	if ok {
		t.Fatalf("should fail the check but it passed")
	}

	expectedFields := map[string]string{
		"args":    "provided args contains invalid chars",
		"command": "command is not supported in this system",
	}
	if !reflect.DeepEqual(fields, expectedFields) {
		t.Logf("expected \n%+v\n got \n%+v\n", expectedFields, fields)
		t.Fatal("expected the returned results fields to be the same as expected results fields")
	}
}
