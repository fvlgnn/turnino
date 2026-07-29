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
	ErrPolicyResetRequired = errors.New("the current number is outside the new policy; confirm reset to continue")
)

type NumberingMode string

const (
	NumberingModeFixed       NumberingMode = "fixed"
	NumberingModeProgressive NumberingMode = "progressive"
)

type Queue struct {
	ID               int64
	Name             string
	NumberingMode    NumberingMode
	Prefix           string
	AlphabeticStart  string
	AlphabeticEnd    string
	Digits           int
	MinValue         int
	MaxValue         int
	CurrentPosition  int
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
	CallKindPrevious  CallKind = "previous"
	CallKindManual    CallKind = "manual"
	CallKindRecall    CallKind = "recall"
)

type Call struct {
	ID        int64
	QueueID   int64
	CounterID int64
	Number    string
	Position  int
	Kind      CallKind
	CalledAt  time.Time
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
	NumberingMode    NumberingMode
	Prefix           string
	AlphabeticStart  string
	AlphabeticEnd    string
	Digits           int
	MinValue         int
	MaxValue         int
	AllowCustomCalls bool
}

var (
	prefixPattern   = regexp.MustCompile(`^[A-Z]{0,3}$`)
	alphabetPattern = regexp.MustCompile(`^[A-Z]{1,2}$`)
)

func (p *CreateQueueParams) NormalizeAndValidate() error {
	p.Name = strings.TrimSpace(p.Name)
	if p.NumberingMode == "" {
		p.NumberingMode = NumberingModeFixed
	}
	p.Prefix = strings.ToUpper(strings.TrimSpace(p.Prefix))
	p.AlphabeticStart = strings.ToUpper(strings.TrimSpace(p.AlphabeticStart))
	p.AlphabeticEnd = strings.ToUpper(strings.TrimSpace(p.AlphabeticEnd))

	if p.Name == "" || len(p.Name) > 100 {
		return errors.New("queue name must contain between 1 and 100 characters")
	}
	switch p.NumberingMode {
	case NumberingModeFixed:
		if !prefixPattern.MatchString(p.Prefix) {
			return errors.New("queue prefix must contain up to 3 ASCII letters")
		}
		p.AlphabeticStart, p.AlphabeticEnd = "", ""
	case NumberingModeProgressive:
		if p.Prefix != "" {
			return errors.New("a progressive queue cannot have a fixed prefix")
		}
		if !alphabetPattern.MatchString(p.AlphabeticStart) ||
			!alphabetPattern.MatchString(p.AlphabeticEnd) ||
			len(p.AlphabeticStart) != len(p.AlphabeticEnd) {
			return errors.New("alphabetic range must use one or two ASCII letters of equal length")
		}
		if alphabetIndex(p.AlphabeticStart) > alphabetIndex(p.AlphabeticEnd) {
			return errors.New("invalid alphabetic range")
		}
	default:
		return errors.New("queue numbering mode must be fixed or progressive")
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

func (p CreateQueueParams) SequenceSize() int {
	numericSize := p.MaxValue - p.MinValue + 1
	if p.NumberingMode != NumberingModeProgressive {
		return numericSize
	}
	return (alphabetIndex(p.AlphabeticEnd) - alphabetIndex(p.AlphabeticStart) + 1) * numericSize
}

func (p CreateQueueParams) FormatPosition(position int) (string, error) {
	if position < 0 || position >= p.SequenceSize() {
		return "", ErrSequenceOutOfRange
	}
	numericSize := p.MaxValue - p.MinValue + 1
	value := p.MinValue + position%numericSize
	if p.NumberingMode == NumberingModeProgressive {
		alpha := alphabetString(alphabetIndex(p.AlphabeticStart)+position/numericSize, len(p.AlphabeticStart))
		return FormatNumber(alpha, p.Digits, value), nil
	}
	return FormatNumber(p.Prefix, p.Digits, value), nil
}

func (p CreateQueueParams) ParseNumber(number string) (int, error) {
	number = strings.ToUpper(strings.TrimSpace(number))
	alpha := p.Prefix
	if p.NumberingMode == NumberingModeProgressive {
		width := len(p.AlphabeticStart)
		if len(number) != width+p.Digits {
			return 0, ErrSequenceOutOfRange
		}
		alpha = number[:width]
		if !alphabetPattern.MatchString(alpha) {
			return 0, ErrSequenceOutOfRange
		}
	} else if len(number) != len(p.Prefix)+p.Digits || !strings.HasPrefix(number, p.Prefix) {
		return 0, ErrSequenceOutOfRange
	}
	valueText := number[len(alpha):]
	value := 0
	for _, digit := range valueText {
		if digit < '0' || digit > '9' {
			return 0, ErrSequenceOutOfRange
		}
		value = value*10 + int(digit-'0')
	}
	if value < p.MinValue || value > p.MaxValue {
		return 0, ErrSequenceOutOfRange
	}
	alphaOffset := 0
	if p.NumberingMode == NumberingModeProgressive {
		index := alphabetIndex(alpha)
		start, end := alphabetIndex(p.AlphabeticStart), alphabetIndex(p.AlphabeticEnd)
		if index < start || index > end {
			return 0, ErrSequenceOutOfRange
		}
		alphaOffset = index - start
	}
	return alphaOffset*(p.MaxValue-p.MinValue+1) + value - p.MinValue, nil
}

func (q Queue) Policy() CreateQueueParams {
	return CreateQueueParams{
		Name: q.Name, NumberingMode: q.NumberingMode, Prefix: q.Prefix,
		AlphabeticStart: q.AlphabeticStart, AlphabeticEnd: q.AlphabeticEnd,
		Digits: q.Digits, MinValue: q.MinValue, MaxValue: q.MaxValue,
		AllowCustomCalls: q.AllowCustomCalls,
	}
}

func (q Queue) CurrentNumber() string {
	if q.CurrentPosition < 0 {
		return ""
	}
	number, _ := q.Policy().FormatPosition(q.CurrentPosition)
	return number
}

func (q Queue) FirstNumber() string {
	number, _ := q.Policy().FormatPosition(0)
	return number
}

func (q Queue) LastNumber() string {
	number, _ := q.Policy().FormatPosition(q.Policy().SequenceSize() - 1)
	return number
}

func alphabetIndex(value string) int {
	index := 0
	for _, letter := range value {
		index = index*26 + int(letter-'A')
	}
	return index
}

func alphabetString(index, width int) string {
	letters := make([]byte, width)
	for position := width - 1; position >= 0; position-- {
		letters[position] = byte('A' + index%26)
		index /= 26
	}
	return string(letters)
}
