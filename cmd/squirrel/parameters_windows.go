//go:build windows

package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/core"
)

type modelParameterDraft struct {
	Temperature, TopP, MaxOutputTokens, PresencePenalty, FrequencyPenalty, Seed string
}

func parameterDraft(c core.Config) modelParameterDraft {
	floatText := func(value *float64) string {
		if value == nil {
			return ""
		}
		return strconv.FormatFloat(*value, 'g', -1, 64)
	}
	intText := func(value *int) string {
		if value == nil {
			return ""
		}
		return strconv.Itoa(*value)
	}
	return modelParameterDraft{Temperature: floatText(c.Temperature), TopP: floatText(c.TopP), MaxOutputTokens: intText(c.MaxOutputTokens), PresencePenalty: floatText(c.PresencePenalty), FrequencyPenalty: floatText(c.FrequencyPenalty), Seed: intText(c.Seed)}
}

func (d modelParameterDraft) apply(c core.Config) (core.Config, error) {
	for _, field := range []struct {
		name, text string
		out        **float64
	}{
		{tr("温度"), d.Temperature, &c.Temperature}, {"Top P", d.TopP, &c.TopP},
		{tr("存在惩罚"), d.PresencePenalty, &c.PresencePenalty}, {tr("频率惩罚"), d.FrequencyPenalty, &c.FrequencyPenalty},
	} {
		*field.out = nil
		if text := strings.TrimSpace(field.text); text != "" {
			value, err := strconv.ParseFloat(text, 64)
			if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
				return c, fmt.Errorf(tr("%s需要填写有效数字，留空使用默认值"), field.name)
			}
			*field.out = &value
		}
	}
	for _, field := range []struct {
		name, text string
		out        **int
	}{{tr("输出上限"), d.MaxOutputTokens, &c.MaxOutputTokens}, {"Seed", d.Seed, &c.Seed}} {
		*field.out = nil
		if text := strings.TrimSpace(field.text); text != "" {
			value, err := strconv.Atoi(text)
			if err != nil {
				return c, fmt.Errorf(tr("%s需要填写整数，留空使用默认值"), field.name)
			}
			*field.out = &value
		}
	}
	return c, c.Check()
}

func reasoningSelect(c *ui.Context, value *string) ui.Element {
	labels := []string{tr("默认"), tr("关闭"), tr("极低"), tr("低"), tr("中"), tr("高"), tr("极高"), tr("最高")}
	display := *value
	for i, v := range core.ReasoningEfforts {
		if v == *value {
			display = labels[i]
		}
	}
	b := formSelect(c, &display, labels).Label(tr("思考等级")).Tooltip(tr("按模型支持选择；默认不发送 reasoning_effort"))
	if b.Changed() {
		for i, label := range labels {
			if display == label {
				*value = core.ReasoningEfforts[i]
			}
		}
	}
	return b
}

func (a *application) modelSettings(c *ui.Context) {
	s := uiSpacing.Main
	formRow(c, tr("思考等级"), func() { reasoningSelect(c, &a.draft.ReasoningEffort).Grow(1) })
	formRow(c, tr("温度"), func() {
		field(c, &a.parameters.Temperature, tr("默认")).Label(tr("温度")).Tooltip(tr("0–2；留空不发送 temperature")).Grow(1)
	})
	formRow(c, tr("输出上限"), func() {
		ui.Row(c).Grow(1).Gap(s.InlineGap).AlignItems(ui.Center).Children(func() {
			field(c, &a.parameters.MaxOutputTokens, tr("默认")).Label(tr("最大输出 Token")).Tooltip(tr("正整数；留空使用模型默认上限")).Grow(1)
			formSelect(c, &a.draft.TokenLimitField, []string{"max_tokens", "max_completion_tokens"}).Label(tr("输出上限格式")).Tooltip(tr("按供应商支持选择上限字段"))
		})
	})
	ui.Row(c).Margin(0, -uiSpacing.Controls.ButtonX).Children(func() {
		disclosure(c, tr("更多参数"), &a.moreModelParameters)
	})
	if a.moreModelParameters {
		formRow(c, "Top P", func() {
			field(c, &a.parameters.TopP, tr("默认")).Label("Top P").Tooltip(tr("0–1；通常只调整温度或 Top P 中的一项")).Grow(1)
		})
		formRow(c, tr("频率惩罚"), func() {
			field(c, &a.parameters.FrequencyPenalty, tr("默认")).Label(tr("频率惩罚")).Tooltip(tr("-2–2；正值降低重复内容的频率")).Grow(1)
		})
		formRow(c, tr("存在惩罚"), func() {
			field(c, &a.parameters.PresencePenalty, tr("默认")).Label(tr("存在惩罚")).Tooltip(tr("-2–2；正值鼓励使用尚未出现的内容")).Grow(1)
		})
		formRow(c, "Seed", func() {
			field(c, &a.parameters.Seed, tr("默认")).Label(tr("随机种子")).Tooltip(tr("整数；需要模型支持，不保证完全复现")).Grow(1)
		})
		ui.Row(c).Margin(0, -uiSpacing.Controls.ButtonX).Children(func() {
			if textButton(c, tr("重置参数")).Clicked() {
				a.parameters = modelParameterDraft{}
				a.draft.ReasoningEffort = "default"
				a.draft.TokenLimitField = "max_tokens"
				a.status = tr("参数已重置，保存后生效")
			}
		})
	}
}
