package repl

// ==================== 辅助函数 ====================
func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
