package core

import (
	"fmt"
	"github.com/simiooo/squirrel_translate/internal/i18n"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
)

var Validate = validator.New(validator.WithRequiredStructEnabled())

type Config struct {
	UILanguage        string   `json:"ui_language,omitempty" validate:"omitempty,oneof=system zh-CN en"`
	Endpoint          string   `json:"endpoint" validate:"required,url"`
	Model             string   `json:"model" validate:"required,max=160"`
	PrimaryLanguage   string   `json:"primary_language" validate:"required"`
	SecondaryLanguage string   `json:"secondary_language" validate:"required"`
	Shortcut          string   `json:"shortcut" validate:"required"`
	TimeoutSeconds    int      `json:"timeout_seconds" validate:"min=10,max=600"`
	IncludeContext    bool     `json:"include_context"`
	AutoCapture       bool     `json:"auto_capture"`
	StructuredMode    string   `json:"structured_mode" validate:"oneof=prompt json_object json_schema"`
	ReasoningEffort   string   `json:"reasoning_effort" validate:"oneof=default none minimal low medium high xhigh max"`
	Temperature       *float64 `json:"temperature,omitempty" validate:"omitempty,gte=0,lte=2"`
	TopP              *float64 `json:"top_p,omitempty" validate:"omitempty,gte=0,lte=1"`
	MaxOutputTokens   *int     `json:"max_output_tokens,omitempty" validate:"omitempty,min=1"`
	TokenLimitField   string   `json:"token_limit_field" validate:"oneof=max_tokens max_completion_tokens"`
	PresencePenalty   *float64 `json:"presence_penalty,omitempty" validate:"omitempty,gte=-2,lte=2"`
	FrequencyPenalty  *float64 `json:"frequency_penalty,omitempty" validate:"omitempty,gte=-2,lte=2"`
	Seed              *int     `json:"seed,omitempty"`
	ExcludedApps      string   `json:"excluded_apps"`
	Appearance        string   `json:"appearance,omitempty" validate:"omitempty,oneof=跟随系统 浅色 深色"`
}

func DefaultConfig() Config {
	return Config{Endpoint: "https://api.deepseek.com", Model: "deepseek-chat", PrimaryLanguage: "简体中文", SecondaryLanguage: "English", Shortcut: "Ctrl+Alt+T", TimeoutSeconds: 90, IncludeContext: true, AutoCapture: true, StructuredMode: "prompt", ReasoningEffort: "default", TokenLimitField: "max_tokens", Appearance: "跟随系统"}
}

var ReasoningEfforts = []string{"default", "none", "minimal", "low", "medium", "high", "xhigh", "max"}

func (c Config) Check() error {
	if err := Validate.Struct(c); err != nil {
		if fields, ok := err.(validator.ValidationErrors); ok && len(fields) > 0 {
			if message := map[string]string{
				"Temperature": "温度需要在 0–2 之间", "TopP": "Top P 需要在 0–1 之间",
				"MaxOutputTokens": "输出上限需要是大于 0 的整数", "TokenLimitField": "输出上限参数格式无效",
				"PresencePenalty": "存在惩罚需要在 -2–2 之间", "FrequencyPenalty": "频率惩罚需要在 -2–2 之间",
				"ReasoningEffort": "思考等级无效",
			}[fields[0].Field()]; message != "" {
				return fmt.Errorf("%s", i18n.T(message))
			}
		}
		return fmt.Errorf(i18n.T("请检查必填配置及超时范围（10–600 秒）：%w"), err)
	}
	u, err := url.Parse(c.Endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%s", i18n.T("接口地址必须是 http(s) 基础地址，不含用户名、查询参数或锚点"))
	}
	if c.PrimaryLanguage == c.SecondaryLanguage {
		return fmt.Errorf("%s", i18n.T("首选语言与第二语言需要不同"))
	}
	return nil
}

var Languages = []string{"自动", "简体中文", "繁體中文", "English", "日本語", "한국어", "Français", "Deutsch", "Español", "Русский"}
var Modes = []string{"自动", "单词", "短语", "句子 / 段落"}

func LanguageForLocale(locale string) string {
	locale = strings.ToLower(strings.ReplaceAll(locale, "_", "-"))
	if strings.HasPrefix(locale, "zh") {
		if strings.Contains(locale, "tw") || strings.Contains(locale, "hk") || strings.Contains(locale, "hant") {
			return "繁體中文"
		}
		return "简体中文"
	}
	for code, name := range map[string]string{"en": "English", "ja": "日本語", "ko": "한국어", "fr": "Français", "de": "Deutsch", "es": "Español", "ru": "Русский"} {
		if strings.HasPrefix(locale, code) {
			return name
		}
	}
	return "English"
}

type Rect struct{ X, Y, Width, Height int }
type Selection struct {
	Text    string  `json:"text"`
	Context string  `json:"context,omitempty"`
	Locale  string  `json:"locale,omitempty"`
	Source  string  `json:"source,omitempty"`
	Owner   uintptr `json:"owner,omitempty"`
	Bounds  Rect    `json:"bounds"`
}

func Classify(s string) string {
	s = strings.Trim(strings.TrimSpace(s), "\"'“”‘’()[]")
	if s == "" {
		return "sentence"
	}
	words := 0
	inWord := false
	han := 0
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
			han++
		}
		if strings.ContainsRune("。！？!?；;，,：:\n\r", r) || r == '.' && strings.HasSuffix(s, ".") {
			return "sentence"
		}
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			words++
			inWord = true
		}
	}
	if looksLikeClause(s) {
		return "sentence"
	}
	if han > 0 {
		if utf8.RuneCountInString(s) <= 4 && words == 1 {
			return "word"
		}
		if utf8.RuneCountInString(s) <= 16 && words <= 6 {
			return "phrase"
		}
		return "sentence"
	}
	if words == 1 && utf8.RuneCountInString(s) <= 64 {
		return "word"
	}
	if words >= 2 && words <= 8 && utf8.RuneCountInString(s) <= 120 {
		return "phrase"
	}
	return "sentence"
}

// A fast, deliberately bounded hint, not a multilingual grammar parser.
// Ambiguous fragments can be corrected in the translation window without
// paying for a separate classification request.
func looksLikeClause(s string) bool {
	fields := strings.Fields(strings.ToLower(s))
	if len(fields) > 1 {
		first := strings.ReplaceAll(fields[0], "’", "'")
		for _, subject := range []string{"i", "you", "he", "she", "it", "we", "they", "i'm", "you're", "he's", "she's", "it's", "we're", "they're", "i've", "we've", "they've", "i'll", "you'll", "we'll"} {
			if first == subject {
				return true
			}
		}
		for _, word := range fields[1:] {
			if word == "is" || word == "are" || word == "was" || word == "were" {
				return true
			}
		}
	}
	if utf8.RuneCountInString(s) > 2 {
		for _, subject := range []string{"我", "你", "他", "她", "它", "咱们"} {
			if strings.HasPrefix(s, subject) {
				return true
			}
		}
	}
	return false
}

func IsLexical(mode string) bool { return mode == "word" || mode == "phrase" }

func ResolveMode(text, choice string) string {
	switch choice {
	case "单词":
		return "word"
	case "短语":
		return "phrase"
	case "句子 / 段落":
		return "sentence"
	case "单词 / 短语": // Previous UI and stored sessions.
		if Classify(text) == "word" {
			return "word"
		}
		return "phrase"
	default:
		return Classify(text)
	}
}

func ModeLabel(mode string) string {
	switch mode {
	case "word":
		return "单词"
	case "phrase":
		return "短语"
	default:
		return "句子 / 段落"
	}
}

func SourceLanguage(s string) string {
	for _, r := range s {
		if unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
			return "ja"
		}
		if unicode.Is(unicode.Hangul, r) {
			return "ko"
		}
	}
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return "zh"
		}
		if unicode.Is(unicode.Cyrillic, r) {
			return "ru"
		}
	}
	// Latin text is ambiguous (especially short words). The model supplies the
	// exact language; this hint is used only for choosing a useful default target.
	return "latin"
}

func Target(text, requested string, cfg Config) string {
	if requested != "自动" {
		return requested
	}
	lang := SourceLanguage(text)
	primary := cfg.PrimaryLanguage
	same := (lang == "zh" && strings.Contains(primary, "中文")) || (lang == "ja" && primary == "日本語") || (lang == "ko" && primary == "한국어") || (lang == "latin" && primary == "English") || (lang == "ru" && primary == "Русский")
	if same {
		return cfg.SecondaryLanguage
	}
	return primary
}

type Example struct {
	Text        string `json:"text" validate:"required,max=2000"`
	Translation string `json:"translation" validate:"required,max=2000"`
}
type Sense struct {
	PartOfSpeech string    `json:"part_of_speech" validate:"required,max=120"`
	Meaning      string    `json:"meaning" validate:"required,max=3000"`
	Usage        string    `json:"usage" validate:"max=1500"`
	InContext    bool      `json:"in_context"`
	Examples     []Example `json:"examples" validate:"max=3,dive"`
}
type Dictionary struct {
	Word           string   `json:"word" validate:"required,max=160"`
	Lemma          string   `json:"lemma" validate:"required,max=160"`
	SourceLanguage string   `json:"source_language" validate:"required,max=40"`
	Pronunciation  string   `json:"pronunciation" validate:"max=200"`
	Forms          []string `json:"forms" validate:"max=16,dive,max=160"`
	Senses         []Sense  `json:"senses" validate:"required,min=1,max=32,dive"`
	Collocations   []string `json:"collocations" validate:"max=16,dive,max=300"`
}

type Entry struct {
	ID             string     `json:"id"`
	Word           string     `json:"word"`
	Kind           string     `json:"kind,omitempty"`
	SourceLanguage string     `json:"source_language"`
	TargetLanguage string     `json:"target_language"`
	Definition     Dictionary `json:"definition"`
	Context        string     `json:"context,omitempty"`
	Source         string     `json:"source,omitempty"`
	Note           string     `json:"note,omitempty"`
	AddedAt        time.Time  `json:"added_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Version-1 files saved before categories remain readable.
func (e Entry) LexicalKind() string {
	if IsLexical(e.Kind) {
		return e.Kind
	}
	if Classify(e.Word) == "word" {
		return "word"
	}
	return "phrase"
}
