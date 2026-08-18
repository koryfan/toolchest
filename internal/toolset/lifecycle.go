package toolset

// CloseAll 停止所有工具的后台任务（应用退出时调用）
func CloseAll() {
	for _, t := range All() {
		if closer, ok := t.(interface{ CloseAll() }); ok {
			closer.CloseAll()
		}
	}
}
