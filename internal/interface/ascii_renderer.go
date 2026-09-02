package repl

import (
	"fmt"
	"strings"

	"astrolex/internal/domain"
	"astrolex/internal/i18n"
)

// ASCIIRenderer 负责渲染 ASCII 火箭发射 UI
type ASCIIRenderer struct {
	// 火箭设计（用于确定级数、载荷等）
	design *domain.RocketDesign
	// 当前遥测数据
	height       float64 // km
	speed        float64 // km/s
	accel        float64 // G
	timeElapsed  float64 // 秒
	deltaVUsed   float64 // m/s
	currentStage int     // 当前级数（从1开始）
	totalStages  int     // 总级数
	// 动画状态
	tailFlameLength int  // 尾焰长度（字符数）
	stageSeparated  bool // 是否已分离当前级
	// 终端尺寸（近似）
	termWidth  int
	termHeight int
}

// NewASCIIRenderer 创建新的 ASCII 渲染器
func NewASCIIRenderer(design *domain.RocketDesign, totalStages int) *ASCIIRenderer {
	return &ASCIIRenderer{
		design:          design,
		totalStages:     totalStages,
		currentStage:    1,
		tailFlameLength: 6,
		termWidth:       80,
		termHeight:      24,
	}
}

// Update 更新遥测数据
func (r *ASCIIRenderer) Update(height, speed, accel, timeElapsed, deltaVUsed float64) {
	r.height = height
	r.speed = speed
	r.accel = accel
	r.timeElapsed = timeElapsed
	r.deltaVUsed = deltaVUsed
}

// SetStage 设置当前级数（用于级分离时更新）
func (r *ASCIIRenderer) SetStage(stage int) {
	r.currentStage = stage
}

// SetTailFlame 设置尾焰长度（随推力变化）
func (r *ASCIIRenderer) SetTailFlame(length int) {
	if length < 0 {
		length = 0
	}
	if length > 10 {
		length = 10
	}
	r.tailFlameLength = length
}

// SetStageSeparated 标记级已分离
func (r *ASCIIRenderer) SetStageSeparated(separated bool) {
	r.stageSeparated = separated
}

// Render 渲染完整画面
func (r *ASCIIRenderer) Render() string {
	var sb strings.Builder

	// 清屏并移动光标到左上角
	sb.WriteString("\033[2J\033[H")

	// 绘制顶部边框和遥测面板
	sb.WriteString(r.renderTelemetry())

	// 绘制火箭
	sb.WriteString(r.renderRocket())

	// 绘制底部状态行
	sb.WriteString(r.renderStatusLine())

	return sb.String()
}

// renderTelemetry 渲染侧边遥测面板
func (r *ASCIIRenderer) renderTelemetry() string {
	var sb strings.Builder

	sb.WriteString("╔════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString(fmt.Sprintf("║                      %-22s                     ║\n", i18n.T("launch_status_title")))
	sb.WriteString("╠════════════════════════════════════════════════════════════════╣\n")
	sb.WriteString(fmt.Sprintf("║  %-8s %8.1f km  |   %-6s %7.2f km/s            ║\n",
		i18n.T("telemetry_altitude"), r.height, i18n.T("telemetry_velocity"), r.speed))
	sb.WriteString(fmt.Sprintf("║  %-8s %6.2f G   |   %-6s %7.1f s              ║\n",
		i18n.T("telemetry_accel"), r.accel, i18n.T("telemetry_time"), r.timeElapsed))
	sb.WriteString(fmt.Sprintf("║  %-8s %d/%d        |   %-6s %8.0f m/s      ║\n",
		i18n.T("telemetry_stage"), r.currentStage, r.totalStages,
		i18n.T("telemetry_remaining_dv"), r.design.DeltaV-r.deltaVUsed))
	sb.WriteString("╚════════════════════════════════════════════════════════════════╝\n")

	return sb.String()
}

// renderRocket 渲染 ASCII 火箭（含尾焰）
func (r *ASCIIRenderer) renderRocket() string {
	var sb strings.Builder

	// 计算火箭的垂直位置（居中）
	rocketHeight := 10 + r.totalStages*2 // 估算
	startRow := 12 - rocketHeight/2
	if startRow < 1 {
		startRow = 1
	}

	// 火箭各部分的字符画
	// 从上到下：载荷段、级段、引擎、尾焰

	// 载荷段（顶部）
	payloadLines := []string{
		"         /\\",
		"        /  \\",
		"       /    \\",
		"      /      \\",
		"     /  " + i18n.T("ascii_payload") + "  \\",
		"    /  " + i18n.T("ascii_satellite_bay") + " \\",
		"   /____________\\",
	}

	// 级段（每个级包含燃料箱和引擎）
	stageLines := r.renderStages()

	// 尾焰
	flameLines := r.renderFlame()

	// 组合所有部分，居中
	allLines := append(payloadLines, stageLines...)
	allLines = append(allLines, flameLines...)

	for _, line := range allLines {
		// 简单居中（假设终端宽度80）
		padding := (80 - len(line)) / 2
		if padding < 0 {
			padding = 0
		}
		sb.WriteString(strings.Repeat(" ", padding))
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderStages 渲染各级（从上到下）
func (r *ASCIIRenderer) renderStages() []string {
	var lines []string

	for i := 1; i <= r.totalStages; i++ {
		// 判断当前级是否已分离
		if r.stageSeparated && i == r.currentStage-1 {
			// 已分离的级用虚线表示
			lines = append(lines, "   |----------|")
			lines = append(lines, "   | " + i18n.T("ascii_separated") + " |")
			lines = append(lines, "   |----------|")
			continue
		}
		// 普通级
		stageNum := fmt.Sprintf(" " + i18n.T("ascii_stage") + " %d", i)
		lines = append(lines, "   __________")
		lines = append(lines, "  |          |")
		lines = append(lines, fmt.Sprintf("  |%s |", stageNum))
		lines = append(lines, "  |          |")
		lines = append(lines, "  |  ██████  |")
		lines = append(lines, "  |  ██████  |")
		lines = append(lines, "  |  ██████  |")
		if i == r.totalStages {
			// 最后一级包含引擎
			lines = append(lines, "  | " + i18n.T("ascii_engine") + " |")
			lines = append(lines, "  |  ████   |")
		}
		lines = append(lines, "  |__________|")
	}

	return lines
}

// renderFlame 渲染尾焰（随推力变化）
func (r *ASCIIRenderer) renderFlame() []string {
	var lines []string

	if r.tailFlameLength == 0 {
		lines = append(lines, "    +")
		return lines
	}

	// 尾焰字符：从密集到稀疏
	flameChars := []string{"█", "▓", "▒", "░", " "}
	flameLen := r.tailFlameLength

	// 构建尾焰线条（从引擎底部向下延伸）
	for i := 0; i < flameLen; i++ {
		// 尾焰宽度随距离增加而变宽，然后变窄
		width := 2 + i*2
		if width > 10 {
			width = 10 - (i - 4)
		}
		if width < 2 {
			width = 2
		}
		charIdx := i % len(flameChars)
		char := flameChars[charIdx]
		line := strings.Repeat(" ", 10-width/2) + strings.Repeat(char, width)
		lines = append(lines, line)
	}

	return lines
}

// renderStatusLine 渲染底部状态行
func (r *ASCIIRenderer) renderStatusLine() string {
	status := "🚀 " + i18n.T("ascii_status_normal")
	if r.stageSeparated {
		status = "🔧 " + i18n.T("ascii_status_separating")
	}
	if r.tailFlameLength == 0 {
		status = "🛑 " + i18n.T("ascii_status_shutdown")
	}
	return fmt.Sprintf("\n\n  %s  |  %s: %d/%d", status, i18n.T("telemetry_stage"), r.currentStage, r.totalStages)
}

// Clear 清屏
func (r *ASCIIRenderer) Clear() {
	fmt.Print("\033[2J\033[H")
}
