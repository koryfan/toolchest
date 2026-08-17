package transform

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// 默认日期格式（按优先级尝试）
var defaultDateLayouts = []string{
	time.RFC3339,                    // 2023-11-14T22:13:20Z
	"2006-01-02 15:04:05 -0700 MST", // 带数字时区
	"2006-01-02 15:04:05 MST",       // 带时区缩写（如 UTC/CST）
	"2006-01-02 15:04:05",           // 常见带时间
	"2006/01/02 15:04:05",           // 斜杠分隔带时间
	"2006-01-02",                    // 只日期
	"2006/01/02",                    // 斜杠只日期
}

// TimeParser 是时间戳解析器，持有配置和输入，Parse 后用 Result 取结果
// 参考 JSONTransformer 的封装模式：配置与输入分离，计算与取值分离
type TimeParser struct {
	layouts  []string // 候选日期格式
	timezone *time.Location
	input    string
	result   ParseResult
	parsed   bool // 是否已解析（避免没 Parse 就取结果）
}

// NewTimeParser 用默认配置构造解析器
func NewTimeParser() *TimeParser {
	return &TimeParser{
		layouts:  defaultDateLayouts,
		timezone: time.Local,
	}
}

// SetLayouts 设置候选日期格式（覆盖默认）
func (p *TimeParser) SetLayouts(layouts []string) {
	p.layouts = layouts
	p.parsed = false
}

// SetTimezone 设置时区（nil 表示本地）
func (p *TimeParser) SetTimezone(loc *time.Location) {
	p.timezone = loc
	p.parsed = false
}

// SetInput 设置要解析的输入，并标记需重新解析
func (p *TimeParser) SetInput(input string) {
	p.input = input
	p.parsed = false
}

// Parse 执行解析，结果存内部，后续用 Result 取
func (p *TimeParser) Parse() {
	p.result = p.doParse()
	p.parsed = true
}

// Result 返回最近一次 Parse 的结果
func (p *TimeParser) Result() ParseResult {
	if !p.parsed {
		p.Parse()
	}
	return p.result
}

// doParse 实际解析逻辑
func (p *TimeParser) doParse() ParseResult {
	s := strings.TrimSpace(p.input)
	if s == "" {
		return ParseResult{} // 空输入：空结果，不算错
	}
	// 纯数字：判断秒/毫秒
	if isAllDigits(s) {
		return p.parseDigits(s)
	}
	// 含字母：当日期字符串解析
	return p.parseDateString(s)
}

// isAllDigits 判断是否全是数字
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// parseDigits 处理纯数字输入（时间戳）
func (p *TimeParser) parseDigits(s string) ParseResult {
	var sec int64
	var err error
	switch len(s) {
	case 10:
		// 秒级时间戳
		sec, err = strconv.ParseInt(s, 10, 64)
	case 13:
		// 毫秒级时间戳
		var ms int64
		ms, err = strconv.ParseInt(s, 10, 64)
		sec = ms / 1000
	default:
		// 其它位数：尝试当秒解析
		sec, err = strconv.ParseInt(s, 10, 64)
	}
	if err != nil {
		return ParseResult{Error: "数字超出范围，无法解析"}
	}
	return p.buildResult(time.Unix(sec, 0))
}

// parseDateString 尝试用预设格式解析日期字符串
func (p *TimeParser) parseDateString(s string) ParseResult {
	for _, layout := range p.layouts {
		if t, err := time.ParseInLocation(layout, s, p.timezone); err == nil {
			return p.buildResult(t)
		}
	}
	return ParseResult{Error: "无法识别的格式"}
}

// buildResult 从 time.Time 填充 ParseResult
func (p *TimeParser) buildResult(t time.Time) ParseResult {
	localT := t.In(p.timezone)
	return ParseResult{
		Valid:        true,
		LocalTime:    localT.Format("2006-01-02 15:04:05 -0700 MST"),
		UTCTime:      t.UTC().Format("2006-01-02 15:04:05 UTC"),
		RelativeTime: relativeTime(t),
		Weekday:      "星期" + weekdayChinese(t.Weekday()),
		UnixSecond:   strconv.FormatInt(t.Unix(), 10),
		UnixMilli:    strconv.FormatInt(t.UnixMilli(), 10),
	}
}

// relativeTime 计算相对现在的描述
func relativeTime(t time.Time) string {
	diff := time.Since(t)
	abs := diff
	if abs < 0 {
		abs = -abs
	}
	// 差值小于 60 秒
	if abs < time.Minute {
		if diff >= 0 {
			return "刚刚"
		}
		return "即将"
	}
	suffix := "前"
	if diff < 0 {
		suffix = "后"
	}
	switch {
	case abs < time.Hour:
		return fmt.Sprintf("%d 分钟%s", int(abs.Minutes()), suffix)
	case abs < 24*time.Hour:
		return fmt.Sprintf("%d 小时%s", int(abs.Hours()), suffix)
	case abs < 30*24*time.Hour:
		return fmt.Sprintf("%d 天%s", int(abs.Hours()/24), suffix)
	case abs < 365*24*time.Hour:
		return fmt.Sprintf("%d 个月%s", int(abs.Hours()/24/30), suffix)
	default:
		return fmt.Sprintf("%d 年%s", int(abs.Hours()/24/365), suffix)
	}
}

// weekdayChinese 把英文星期转中文数字
func weekdayChinese(w time.Weekday) string {
	names := []string{"日", "一", "二", "三", "四", "五", "六"}
	return names[int(w)]
}

// ParseResult 是时间戳解析的结果
type ParseResult struct {
	Valid        bool   // 是否成功解析
	LocalTime    string // 本地时间，如 "2023-11-15 06:13:20 +0800 CST"
	UTCTime      string // UTC 时间
	RelativeTime string // 相对现在，如 "2 天前"
	Weekday      string // 星期几（中文）
	UnixSecond   string // 秒级时间戳
	UnixMilli    string // 毫秒级时间戳
	Error        string // 无法识别时的错误提示
}
