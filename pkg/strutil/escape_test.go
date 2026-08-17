package strutil

import "testing"

func TestEscape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"普通字符串", `hello`, `"hello"`},
		{"带换行", "hello\nworld", `"hello\nworld"`},
		{"带双引号", `he said "hi"`, `"he said \"hi\""`},
		{"带中文", `你好`, `"你好"`},
		{"空字符串", ``, `""`},
		{"带反斜杠", `a\b`, `"a\\b"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Escape(tt.input); got != tt.want {
				t.Errorf("Escape() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnescape(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"带引号普通", `"hello"`, `hello`, false},
		{"带引号换行", `"hello\nworld"`, "hello\nworld", false},
		{"不带引号自动补", `hello`, `hello`, false},
		{"带转义引号", `"he said \"hi\""`, `he said "hi"`, false},
		{"空字符串", ``, ``, false},
		{"非法转义", `"\xyz"`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Unescape(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unescape() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Unescape() = %q, want %q", got, tt.want)
			}
		})
	}
}
