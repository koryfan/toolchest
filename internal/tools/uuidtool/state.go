package uuidtool

// State 持久化状态：版本与数量选中项（重启即恢复）
type State struct {
	Version string `yaml:"version" json:"version"` // v4 / v7
	Count   int    `yaml:"count" json:"count"`     // 1..10
}

// DefaultState 默认 v7（时间有序）、1 个
func DefaultState() State {
	return State{Version: "v7", Count: 1}
}

// normalize 校正状态：非法版本回 v7，数量夹到 1..10
func (s *State) normalize() {
	if s.Version != "v4" && s.Version != "v7" {
		s.Version = "v7"
	}
	if s.Count < 1 {
		s.Count = 1
	}
	if s.Count > MaxCount {
		s.Count = MaxCount
	}
}
