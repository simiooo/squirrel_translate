//go:build windows

package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/simiooo/squirrel_translate/internal/core"
)

// Distances are logical pixels (DIPs); line heights are unitless ratios. The JSON
// is the single source of defaults; a debug override may contain only the
// fields being tuned. Geometry such as icon strokes and font sizes is separate.
//
//go:embed theme/spacing.json
var defaultSpacingJSON []byte

type spacingTheme struct {
	Main        mainSpacing        `json:"main"`
	Translation translationSpacing `json:"translation"`
	Controls    controlSpacing     `json:"controls"`
	Dictionary  dictionarySpacing  `json:"dictionary"`
	Vocabulary  vocabularySpacing  `json:"vocabulary"`
	Toolbar     toolbarSpacing     `json:"toolbar"`
}

type mainSpacing struct {
	InsetX     float32 `json:"inset_x" validate:"gte=0,lte=128"`
	InsetY     float32 `json:"inset_y" validate:"gte=0,lte=128"`
	HeaderY    float32 `json:"header_y" validate:"gte=0,lte=128"`
	HeaderGap  float32 `json:"header_gap" validate:"gte=0,lte=128"`
	SectionGap float32 `json:"section_gap" validate:"gte=0,lte=128"`
	RowGap     float32 `json:"row_gap" validate:"gte=0,lte=128"`
	InlineGap  float32 `json:"inline_gap" validate:"gte=0,lte=128"`
	LabelWidth float32 `json:"label_width" validate:"gte=32,lte=256"`
	NavY       float32 `json:"nav_y" validate:"gte=0,lte=64"`
	NavGap     float32 `json:"nav_gap" validate:"gte=0,lte=64"`
}

type translationSpacing struct {
	InsetX            float32 `json:"inset_x" validate:"gte=0,lte=128"`
	HeaderTop         float32 `json:"header_top" validate:"gte=0,lte=128"`
	HeaderBottom      float32 `json:"header_bottom" validate:"gte=0,lte=128"`
	HeaderGap         float32 `json:"header_gap" validate:"gte=0,lte=128"`
	PanelY            float32 `json:"panel_y" validate:"gte=0,lte=128"`
	LoadingY          float32 `json:"loading_y" validate:"gte=0,lte=128"`
	LoadingGap        float32 `json:"loading_gap" validate:"gte=0,lte=128"`
	ContentY          float32 `json:"content_y" validate:"gte=0,lte=128"`
	FooterY           float32 `json:"footer_y" validate:"gte=0,lte=128"`
	ActionGap         float32 `json:"action_gap" validate:"gte=0,lte=128"`
	GripTop           float32 `json:"grip_top" validate:"gte=0,lte=32"`
	BodyMinHeight     float32 `json:"body_min_height" validate:"gte=32,lte=256"`
	ParagraphGap      float32 `json:"paragraph_gap" validate:"gt=0,lte=128"`
	ListItemGap       float32 `json:"list_item_gap" validate:"gt=0,lte=128"`
	BodyLineHeight    float32 `json:"body_line_height" validate:"gte=1,lte=3"`
	LoadingLineHeight float32 `json:"loading_line_height" validate:"gte=1,lte=3"`
}

type controlSpacing struct {
	Unit            float32 `json:"unit" validate:"gt=0,lte=16"`
	ButtonX         float32 `json:"button_x" validate:"gte=0,lte=64"`
	ButtonY         float32 `json:"button_y" validate:"gte=0,lte=64"`
	CompactY        float32 `json:"compact_y" validate:"gte=0,lte=64"`
	IconGap         float32 `json:"icon_gap" validate:"gte=0,lte=64"`
	IconButtonSize  float32 `json:"icon_button_size" validate:"gte=24,lte=64"`
	InputX          float32 `json:"input_x" validate:"gte=0,lte=64"`
	InputY          float32 `json:"input_y" validate:"gte=0,lte=64"`
	ToggleX         float32 `json:"toggle_x" validate:"gte=0,lte=64"`
	ToggleY         float32 `json:"toggle_y" validate:"gte=0,lte=64"`
	ToggleSize      float32 `json:"toggle_size" validate:"gte=10,lte=24"`
	ToggleGap       float32 `json:"toggle_gap" validate:"gte=0,lte=64"`
	ToggleMinHeight float32 `json:"toggle_min_height" validate:"gte=20,lte=64"`
	SelectX         float32 `json:"select_x" validate:"gte=0,lte=64"`
	SelectY         float32 `json:"select_y" validate:"gte=0,lte=64"`
	SelectGap       float32 `json:"select_gap" validate:"gte=0,lte=64"`
	SelectMinWidth  float32 `json:"select_min_width" validate:"gte=64,lte=256"`
	MenuInset       float32 `json:"menu_inset" validate:"gte=0,lte=64"`
	MenuOffset      float32 `json:"menu_offset" validate:"gte=0,lte=64"`
	MenuItemX       float32 `json:"menu_item_x" validate:"gte=0,lte=64"`
	MenuItemY       float32 `json:"menu_item_y" validate:"gte=0,lte=64"`
}

type dictionarySpacing struct {
	MetadataGap float32 `json:"metadata_gap" validate:"gte=0,lte=128"`
	SectionGap  float32 `json:"section_gap" validate:"gte=0,lte=128"`
	SenseGap    float32 `json:"sense_gap" validate:"gte=0,lte=128"`
	PairGap     float32 `json:"pair_gap" validate:"gte=0,lte=128"`
	DetailGap   float32 `json:"detail_gap" validate:"gte=0,lte=128"`
}

type vocabularySpacing struct {
	EntryY    float32 `json:"entry_y" validate:"gte=0,lte=128"`
	DetailGap float32 `json:"detail_gap" validate:"gte=0,lte=128"`
}

type toolbarSpacing struct {
	InsetX float32 `json:"inset_x" validate:"gte=0,lte=64"`
	InsetY float32 `json:"inset_y" validate:"gte=0,lte=64"`
	Gap    float32 `json:"gap" validate:"gte=0,lte=64"`
	GripX  float32 `json:"grip_x" validate:"gte=0,lte=64"`
}

var uiSpacing = mustDefaultSpacing()

func mustDefaultSpacing() spacingTheme {
	s, err := readSpacing(nil)
	if err != nil {
		panic(err) // Invalid embedded defaults are a build-time authoring error.
	}
	return s
}

func readSpacing(override []byte) (spacingTheme, error) {
	var s spacingTheme
	decode := func(data []byte) error {
		if data = bytes.TrimSpace(data); len(data) == 0 || data[0] != '{' {
			return fmt.Errorf("spacing theme must be a JSON object")
		}
		d := json.NewDecoder(bytes.NewReader(data))
		d.DisallowUnknownFields()
		if err := d.Decode(&s); err != nil {
			return err
		}
		if err := d.Decode(new(any)); err != io.EOF {
			return fmt.Errorf("spacing theme must contain one JSON object")
		}
		return nil
	}
	if err := decode(defaultSpacingJSON); err != nil {
		return s, err
	}
	if override != nil {
		if err := decode(override); err != nil {
			return s, err
		}
	}
	return s, core.Validate.Struct(s)
}

func configureSpacing(path string) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s, err := readSpacing(data)
	if err != nil {
		return fmt.Errorf(tr("间距主题 %s 无效：%w"), path, err)
	}
	uiSpacing = s
	return nil
}

// Keep the native toolbar bounds in step with its configurable content.
func toolbarSize() (int, int) {
	s, controls := uiSpacing.Toolbar, uiSpacing.Controls
	w := 2*s.InsetX + 16 + 2*s.GripX + 2*s.Gap + 28 + 2*controls.ButtonX + controls.IconButtonSize
	h := 2*s.InsetY + max(controls.IconButtonSize, 18+2*controls.ButtonY)
	return int(math.Ceil(float64(w))), int(math.Ceil(float64(h)))
}
