package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrQueueNotFound       = errors.New("queue not found")
	ErrCounterNotFound     = errors.New("counter not found")
	ErrCounterUnavailable  = errors.New("counter not found or disabled")
	ErrCustomCallsDisabled = errors.New("custom calls are disabled for this queue")
	ErrSequenceOutOfRange  = errors.New("number is outside the queue range")
	ErrNoPreviousCall      = errors.New("no previous call for this queue and counter")
)

type Queue struct {
	ID               int64
	Name             string
	Prefix           string
	Digits           int
	MinValue         int
	MaxValue         int
	CurrentValue     int
	AllowCustomCalls bool
	CreatedAt        time.Time
}

type Counter struct {
	ID        int64
	Name      string
	Enabled   bool
	CreatedAt time.Time
}

type CallKind string

const (
	CallKindAutomatic CallKind = "automatic"
	CallKindManual    CallKind = "manual"
	CallKindRecall    CallKind = "recall"
)

type Call struct {
	ID            int64
	QueueID       int64
	CounterID     int64
	Number        string
	SequenceValue int
	Kind          CallKind
	CalledAt      time.Time
}

type CallRecord struct {
	Call
	QueueName   string
	CounterName string
}

type Settings struct {
	InstallationName         string
	DefaultLanguage          string
	RecentCallsLimit         int
	SoundEnabled             bool
	HighlightDurationSeconds int
}

func DefaultSettings() Settings {
	return Settings{
		InstallationName:         "Turnino",
		DefaultLanguage:          "en",
		RecentCallsLimit:         10,
		SoundEnabled:             true,
		HighlightDurationSeconds: 5,
	}
}

func (s *Settings) NormalizeAndValidate() error {
	s.InstallationName = strings.TrimSpace(s.InstallationName)
	s.DefaultLanguage = strings.ToLower(strings.TrimSpace(s.DefaultLanguage))
	if s.InstallationName == "" || len(s.InstallationName) > 100 {
		return errors.New("installation name must contain between 1 and 100 characters")
	}
	if s.DefaultLanguage != "en" && s.DefaultLanguage != "it" {
		return errors.New("default language must be en or it")
	}
	if s.RecentCallsLimit < 1 || s.RecentCallsLimit > 100 {
		return errors.New("recent calls limit must be between 1 and 100")
	}
	if s.HighlightDurationSeconds < 1 || s.HighlightDurationSeconds > 30 {
		return errors.New("highlight duration must be between 1 and 30 seconds")
	}
	return nil
}

type CreateQueueParams struct {
	Name             string
	Prefix           string
	Digits           int
	MinValue         int
	MaxValue         int
	AllowCustomCalls bool
}

var prefixPattern = regexp.MustCompile(`^[A-Z]{0,3}$`)

func (p *CreateQueueParams) NormalizeAndValidate() error {
	p.Name = strings.TrimSpace(p.Name)
	p.Prefix = strings.ToUpper(strings.TrimSpace(p.Prefix))

	if p.Name == "" || len(p.Name) > 100 {
		return errors.New("queue name must contain between 1 and 100 characters")
	}
	if !prefixPattern.MatchString(p.Prefix) {
		return errors.New("queue prefix must contain up to 3 ASCII letters")
	}
	if p.Digits < 1 || p.Digits > 6 {
		return errors.New("queue digits must be between 1 and 6")
	}
	if p.MinValue < 0 || p.MaxValue < p.MinValue {
		return errors.New("invalid queue number range")
	}
	limit := 1
	for range p.Digits {
		limit *= 10
	}
	if p.MaxValue >= limit {
		return fmt.Errorf("queue maximum must fit in %d digits", p.Digits)
	}
	return nil
}

func FormatNumber(prefix string, digits, value int) string {
	return fmt.Sprintf("%s%0*d", prefix, digits, value)
}
