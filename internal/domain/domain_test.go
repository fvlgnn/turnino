package domain

import "testing"

func TestCreateQueueParamsNormalizeAndValidate(t *testing.T) {
	params := CreateQueueParams{Name: "  General  ", Prefix: " a ", Digits: 3, MinValue: 1, MaxValue: 999}
	if err := params.NormalizeAndValidate(); err != nil {
		t.Fatalf("NormalizeAndValidate() error = %v", err)
	}
	if params.Name != "General" || params.Prefix != "A" {
		t.Fatalf("normalized params = %#v", params)
	}
	if got := FormatNumber(params.Prefix, params.Digits, params.MinValue); got != "A001" {
		t.Fatalf("FormatNumber() = %q, want A001", got)
	}
}

func TestCreateQueueParamsRejectsValuesOutsideDigits(t *testing.T) {
	params := CreateQueueParams{Name: "General", Digits: 2, MinValue: 0, MaxValue: 100}
	if err := params.NormalizeAndValidate(); err == nil {
		t.Fatal("NormalizeAndValidate() error = nil, want validation error")
	}
}
