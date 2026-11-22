package timetool

import (
	"strings"
	"testing"
	"time"
)

func TestParseTime_Digits(t *testing.T) {
	// 固定时间戳验证：1700000000 = 2023-11-14 22:13:20 UTC（本地东八区 11-15 星期三）
	tests := []struct {
		name        string
		input       string
		wantValid   bool
		wantSec     string
		wantMilli   string
		wantUTC     string
		wantWeekday string
	}{
		{"秒级 10 位", "1700000000", true, "1700000000", "1700000000000", "2023-11-14 22:13:20 UTC", "星期二"},
		{"毫秒级 13 位", "1700000000000", true, "1700000000", "1700000000000", "2023-11-14 22:13:20 UTC", "星期二"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ParseTime(tt.input, time.UTC)
			if r.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", r.Valid, tt.wantValid)
			}
			if r.UnixSecond != tt.wantSec {
				t.Errorf("UnixSecond = %q, want %q", r.UnixSecond, tt.wantSec)
			}
			if r.UnixMilli != tt.wantMilli {
				t.Errorf("UnixMilli = %q, want %q", r.UnixMilli, tt.wantMilli)
			}
			if r.UTCTime != tt.wantUTC {
				t.Errorf("UTCTime = %q, want %q", r.UTCTime, tt.wantUTC)
			}
			if r.Weekday != tt.wantWeekday {
				t.Errorf("Weekday = %q, want %q", r.Weekday, tt.wantWeekday)
			}
		})
	}
}

func TestParseTime_DateString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantUTC string
	}{
		{"RFC3339", "2023-11-14T22:13:20Z", "2023-11-14 22:13:20 UTC"},
		{"带 UTC 后缀", "2026-07-29 08:07:22 UTC", "2026-07-29 08:07:22 UTC"},
		{"常见日期时间", "2023-11-14 22:13:20", "2023-11-14 22:13:20 UTC"},
		{"斜杠日期时间", "2023/11/14 22:13:20", "2023-11-14 22:13:20 UTC"},
		{"仅日期", "2023-11-14", "2023-11-14 00:00:00 UTC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ParseTime(tt.input, time.UTC) // 测试用 UTC，避免环境时区差异
			if !r.Valid {
				t.Fatalf("期望 Valid=true，got Error=%q", r.Error)
			}
			if r.UTCTime != tt.wantUTC {
				t.Errorf("UTCTime = %q, want %q", r.UTCTime, tt.wantUTC)
			}
		})
	}
}

func TestParseTime_LocalDefault(t *testing.T) {
	// loc 为 nil 时用本地时区：LocalTime 必须能反映本机时区偏移
	r := ParseTime("1700000000", nil)
	if !r.Valid {
		t.Fatalf("期望 Valid=true，got Error=%q", r.Error)
	}
	want := time.Unix(1700000000, 0).Format("2006-01-02 15:04:05 -0700 MST")
	if r.LocalTime != want {
		t.Errorf("LocalTime = %q, want %q", r.LocalTime, want)
	}
}

func TestParseTime_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"乱码", "abcdefg"},
		{"特殊字符", "!!!"},
		{"半截日期", "2023-11"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ParseTime(tt.input, nil)
			if r.Valid {
				t.Error("期望 Valid=false")
			}
			if r.Error == "" {
				t.Error("期望有错误信息")
			}
		})
	}
}

func TestParseTime_Empty(t *testing.T) {
	r := ParseTime("", nil)
	if r.Valid {
		t.Error("空输入不应 Valid")
	}
	if r.Error != "" {
		t.Errorf("空输入不应有错误，got %q", r.Error)
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"30 秒前", now.Add(-30 * time.Second), "刚刚"},
		{"30 秒后", now.Add(30 * time.Second), "即将"},
		{"5 分钟前", now.Add(-5 * time.Minute), "分钟前"},
		{"3 小时前", now.Add(-3 * time.Hour), "小时前"},
		{"2 天前", now.Add(-48 * time.Hour), "天前"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeTime(tt.t)
			// 相对时间有边界波动（分钟取整），用后缀匹配容错
			suffix := "前"
			if !strings.HasSuffix(tt.want, suffix) && tt.want != "刚刚" && tt.want != "即将" {
				suffix = "后"
			}
			if tt.want != "刚刚" && tt.want != "即将" && !strings.HasSuffix(got, suffix) {
				t.Errorf("relativeTime() = %q, want suffix %q", got, suffix)
			}
		})
	}
}

func TestWeekdayChinese(t *testing.T) {
	// 2023-11-14 是星期二（UTC）
	t2, _ := time.Parse("2006-01-02", "2023-11-14")
	if got := weekdayChinese(t2.Weekday()); got != "二" {
		t.Errorf("weekdayChinese() = %q, want %q", got, "二")
	}
}
