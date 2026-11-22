package timetool

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// dateLayouts 候选日期格式（按优先级尝试）
var dateLayouts = []string{
	time.RFC3339,                    // 2023-11-14T22:13:20Z
	"2006-01-02 15:04:05 -0700 MST", // 带数字时区
	"2006-01-02 15:04:05 MST",       // 带时区缩写（如 UTC/CST）
	"2006-01-02 15:04:05",           // 常见带时间
	"2006/01/02 15:04:05",           // 斜杠分隔带时间
	"2006-01-02",                    // 只日期
	"2006/01/02",                    // 斜杠只日期
}

// ParseResult 时间解析结果
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

// ParseTime 解析纯数字时间戳（10 位秒 / 13 位毫秒）或日期串；
// loc 为 nil 时用本地时区。空输入返回零值（不算错）。
func ParseTime(input string, loc *time.Location) ParseResult {
	if loc == nil {
		loc = time.Local
	}
	s := strings.TrimSpace(input)
	if s == "" {
		return ParseResult{}
	}
	if isAllDigits(s) {
		return parseDigits(s, loc)
	}
	return parseDateString(s, loc)
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
func parseDigits(s string, loc *time.Location) ParseResult {
	var sec int64
	var err error
	switch len(s) {
	case 10:
		sec, err = strconv.ParseInt(s, 10, 64)
	case 13:
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
	return buildResult(time.Unix(sec, 0), loc)
}

// parseDateString 尝试用候选格式解析日期字符串
func parseDateString(s string, loc *time.Location) ParseResult {
	for _, layout := range dateLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return buildResult(t, loc)
		}
	}
	return ParseResult{Error: "无法识别的格式"}
}

// buildResult 从 time.Time 填充 ParseResult
// buildResult 从 time.Time 填充 ParseResult
func buildResult(t time.Time, loc *time.Location) ParseResult {
	localT := t.In(loc)
	return ParseResult{
		Valid:        true,
		LocalTime:    localT.Format("2006-01-02 15:04:05 -0700 MST"),
		UTCTime:      t.UTC().Format("2006-01-02 15:04:05 UTC"),
		RelativeTime: relativeTime(t),
		Weekday:      "星期" + weekdayChinese(localT.Weekday()),
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
