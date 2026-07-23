package domain

import "testing"

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
