package domain

import (
	"errors"
	"testing"
)

func TestSettingsNormalizeAndValidate(t *testing.T) {
	settings := Settings{
		InstallationName:         "  Front Office  ",
		DefaultLanguage:          "IT",
		RecentCallsLimit:         20,
		SoundEnabled:             true,
		HighlightDurationSeconds: 4,
	}
	if err := settings.NormalizeAndValidate(); err != nil {
		t.Fatalf("NormalizeAndValidate() error = %v", err)
	}
	if settings.InstallationName != "Front Office" || settings.DefaultLanguage != "it" {
		t.Fatalf("normalized settings = %#v", settings)
	}

	for name, mutate := range map[string]func(*Settings){
		"name":      func(s *Settings) { s.InstallationName = "" },
		"language":  func(s *Settings) { s.DefaultLanguage = "fr" },
		"recent":    func(s *Settings) { s.RecentCallsLimit = 0 },
		"highlight": func(s *Settings) { s.HighlightDurationSeconds = 31 },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := DefaultSettings()
			mutate(&invalid)
			if err := invalid.NormalizeAndValidate(); err == nil {
				t.Fatal("NormalizeAndValidate() error = nil, want validation error")
			}
		})
	}
}

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

func TestProgressiveNumberingFormatsParsesAndWrapsRange(t *testing.T) {
	params := CreateQueueParams{
		Name: "General", NumberingMode: NumberingModeProgressive,
		AlphabeticStart: "A", AlphabeticEnd: "C",
		Digits: 2, MinValue: 0, MaxValue: 99,
	}
	if err := params.NormalizeAndValidate(); err != nil {
		t.Fatalf("NormalizeAndValidate() error = %v", err)
	}
	if got := params.SequenceSize(); got != 300 {
		t.Fatalf("SequenceSize() = %d, want 300", got)
	}
	for position, want := range map[int]string{0: "A00", 99: "A99", 100: "B00", 299: "C99"} {
		got, err := params.FormatPosition(position)
		if err != nil || got != want {
			t.Errorf("FormatPosition(%d) = %q, %v; want %q", position, got, err, want)
		}
		parsed, err := params.ParseNumber(want)
		if err != nil || parsed != position {
			t.Errorf("ParseNumber(%q) = %d, %v; want %d", want, parsed, err, position)
		}
	}
	if _, err := params.ParseNumber("D00"); !errors.Is(err, ErrSequenceOutOfRange) {
		t.Fatalf("ParseNumber(D00) error = %v, want ErrSequenceOutOfRange", err)
	}
}

func TestProgressiveNumberingSupportsTwoLetterRanges(t *testing.T) {
	params := CreateQueueParams{
		Name: "General", NumberingMode: NumberingModeProgressive,
		AlphabeticStart: "AA", AlphabeticEnd: "AZ",
		Digits: 3, MinValue: 0, MaxValue: 999,
	}
	if err := params.NormalizeAndValidate(); err != nil {
		t.Fatalf("NormalizeAndValidate() error = %v", err)
	}
	got, err := params.FormatPosition(1000)
	if err != nil || got != "AB000" {
		t.Fatalf("FormatPosition(1000) = %q, %v; want AB000", got, err)
	}
}
