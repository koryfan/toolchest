package services

import (
	"tool-dev/internal/transform"
)

// TimeService 时间戳工具的后端服务
type TimeService struct{}

// Parse 解析输入（秒/毫秒时间戳或日期字符串），返回多格式结果
func (s *TimeService) Parse(input string) transform.ParseResult {
	p := transform.NewTimeParser()
	p.SetInput(input)
	p.Parse()
	return p.Result()
}
