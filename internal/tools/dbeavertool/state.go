package dbeavertool

// State 持久化状态：最近使用目录缓存 + 当前目录（重启即恢复列表）
type State struct {
	RecentDirs []string `yaml:"recentDirs" json:"recentDirs"`
	CurrentDir string   `yaml:"currentDir" json:"currentDir"`
}

// DefaultState 默认状态
func DefaultState() State {
	return State{RecentDirs: []string{}}
}

// pushRecent 目录置顶记忆（去重，最多保留 8 个）
func (s *State) pushRecent(dir string) {
	out := []string{dir}
	for _, d := range s.RecentDirs {
		if d != dir {
			out = append(out, d)
		}
	}
	if len(out) > 8 {
		out = out[:8]
	}
	s.RecentDirs = out
	s.CurrentDir = dir
}
