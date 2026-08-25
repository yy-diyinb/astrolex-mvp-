package repl

import (
	"fmt"
	"strconv"
	"time"

	"astrolex/internal/domain"
)

// ==================== 组装项目管理命令 ====================

// createAssembly 创建组装项目
func (r *Repl) createAssembly(name string) {
	if name == "" {
		fmt.Println("用法: assembly create <项目名称>")
		return
	}
	proj := domain.AssemblyProject{
		ID:          fmt.Sprintf("assem_%d", len(r.game.AssemblyProjects)+1),
		Name:        name,
		Status:      domain.AssemblyPlanning,
		Steps:       []domain.AssemblyStep{},
		CurrentStep: 0,
		CreatedAt:   time.Now(),
	}
	r.game.AssemblyProjects = append(r.game.AssemblyProjects, proj)
	fmt.Printf("✅ 组装项目 '%s' 已创建，ID: %s\n", name, proj.ID)
	fmt.Println("使用 'assembly add <项目ID> <模块ID> <数量>' 添加模块步骤")
}

// addAssemblyStep 添加组装步骤
func (r *Repl) addAssemblyStep(projID, moduleID, quantityStr string) {
	qty, err := strconv.Atoi(quantityStr)
	if err != nil || qty <= 0 {
		fmt.Println("❌ 数量必须为正整数")
		return
	}

	var proj *domain.AssemblyProject
	for i := range r.game.AssemblyProjects {
		if r.game.AssemblyProjects[i].ID == projID {
			proj = &r.game.AssemblyProjects[i]
			break
		}
	}
	if proj == nil {
		fmt.Printf("❌ 未找到项目 '%s'\n", projID)
		return
	}

	if proj.Status == domain.AssemblyCompleted {
		fmt.Printf("❌ 项目 '%s' 已完成，无法添加步骤\n", proj.Name)
		return
	}

	// 验证模块ID是否存在（火箭设计、卫星设计或零件）
	exists := false
	moduleType := "未知"

	for _, d := range r.game.RocketDesigns {
		if d.ID == moduleID {
			exists = true
			moduleType = "火箭"
			break
		}
	}
	if !exists {
		for _, d := range r.game.SatelliteDesigns {
			if d.ID == moduleID {
				exists = true
				moduleType = "卫星"
				break
			}
		}
	}
	if !exists {
		if _, ok := r.game.PartsDB[moduleID]; ok {
			exists = true
			moduleType = "零件"
		}
	}
	if !exists {
		fmt.Printf("❌ 未找到模块设计 '%s'（请检查火箭设计、卫星设计或零件ID）\n", moduleID)
		return
	}

	step := domain.AssemblyStep{
		ID:          len(proj.Steps) + 1,
		Description: fmt.Sprintf("%s %s x%d", moduleType, moduleID, qty),
		ModuleID:    moduleID,
		Quantity:    qty,
		Launched:    false,
		Docked:      false,
	}
	proj.Steps = append(proj.Steps, step)
	fmt.Printf("✅ 已添加步骤 %d: %s\n", step.ID, step.Description)
}

// listAssemblyProjects 列出所有组装项目
func (r *Repl) listAssemblyProjects() {
	if len(r.game.AssemblyProjects) == 0 {
		fmt.Println("📋 没有组装项目")
		return
	}
	fmt.Println("📋 组装项目列表:")
	for _, p := range r.game.AssemblyProjects {
		statusIcon := "⏳"
		switch p.Status {
		case domain.AssemblyPlanning:
			statusIcon = "📝"
		case domain.AssemblyInProgress:
			statusIcon = "🔨"
		case domain.AssemblyPaused:
			statusIcon = "⏸️"
		case domain.AssemblyCompleted:
			statusIcon = "✅"
		case domain.AssemblyFailed:
			statusIcon = "❌"
		}
		progress := 0
		if len(p.Steps) > 0 {
			progress = int(float64(p.CurrentStep) / float64(len(p.Steps)) * 100)
		}
		fmt.Printf("  %s %s: %s (状态: %s, 进度: %d%%, 步骤: %d/%d, 创建: %s)\n",
			statusIcon, p.ID, p.Name, p.Status, progress, p.CurrentStep, len(p.Steps), p.CreatedAt.Format("2006-01-02"))
	}
}

// assemblyStatus 查看项目进度
func (r *Repl) assemblyStatus(projID string) {
	var proj *domain.AssemblyProject
	for i := range r.game.AssemblyProjects {
		if r.game.AssemblyProjects[i].ID == projID {
			proj = &r.game.AssemblyProjects[i]
			break
		}
	}
	if proj == nil {
		fmt.Printf("❌ 未找到项目 '%s'\n", projID)
		return
	}

	fmt.Printf("📋 项目: %s (%s)\n", proj.Name, proj.ID)
	fmt.Printf("   状态: %s\n", proj.Status)
	fmt.Printf("   步骤: %d/%d\n", proj.CurrentStep, len(proj.Steps))
	fmt.Printf("   创建: %s\n", proj.CreatedAt.Format("2006-01-02 15:04:05"))
	if !proj.StartedAt.IsZero() {
		fmt.Printf("   开始: %s\n", proj.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if !proj.CompletedAt.IsZero() {
		fmt.Printf("   完成: %s\n", proj.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	if proj.ProgramID != "" {
		fmt.Printf("   关联程序: %s\n", proj.ProgramID)
	}

	fmt.Println("\n  步骤列表:")
	if len(proj.Steps) == 0 {
		fmt.Println("    (无步骤)")
		return
	}
	for i, step := range proj.Steps {
		statusIcon := "⏳"
		if i < proj.CurrentStep {
			statusIcon = "✅"
		} else if i == proj.CurrentStep {
			statusIcon = "🔨"
		}
		statusText := "等待"
		if i < proj.CurrentStep {
			statusText = "已完成"
		} else if i == proj.CurrentStep {
			statusText = "进行中"
		}
		launchStatus := "未发射"
		if step.Launched {
			launchStatus = "已发射"
		}
		dockStatus := "未对接"
		if step.Docked {
			dockStatus = "已对接"
		}
		fmt.Printf("    %s [%d] %s (状态: %s, 发射: %s, 对接: %s)\n",
			statusIcon, step.ID, step.Description, statusText, launchStatus, dockStatus)
		if step.VesselID != "" {
			fmt.Printf("        航天器: %s\n", step.VesselID)
		}
	}
}

// assemblyStart 开始组装项目
func (r *Repl) assemblyStart(projID string) {
	var proj *domain.AssemblyProject
	for i := range r.game.AssemblyProjects {
		if r.game.AssemblyProjects[i].ID == projID {
			proj = &r.game.AssemblyProjects[i]
			break
		}
	}
	if proj == nil {
		fmt.Printf("❌ 未找到项目 '%s'\n", projID)
		return
	}

	if proj.Status == domain.AssemblyCompleted {
		fmt.Printf("❌ 项目 '%s' 已完成\n", proj.Name)
		return
	}

	if proj.Status == domain.AssemblyInProgress {
		fmt.Printf("⏳ 项目 '%s' 已在执行中\n", proj.Name)
		return
	}

	if len(proj.Steps) == 0 {
		fmt.Printf("❌ 项目 '%s' 没有步骤，请先添加模块\n", proj.Name)
		return
	}

	proj.Status = domain.AssemblyInProgress
	proj.StartedAt = time.Now()
	proj.CurrentStep = 0

	fmt.Printf("🔨 项目 '%s' 开始组装\n", proj.Name)
	fmt.Println("提示: 每完成一个步骤，使用 'assembly step <项目ID>' 推进")
}

// assemblyStep 推进到下一步
func (r *Repl) assemblyStep(projID string) {
	var proj *domain.AssemblyProject
	for i := range r.game.AssemblyProjects {
		if r.game.AssemblyProjects[i].ID == projID {
			proj = &r.game.AssemblyProjects[i]
			break
		}
	}
	if proj == nil {
		fmt.Printf("❌ 未找到项目 '%s'\n", projID)
		return
	}

	if proj.Status != domain.AssemblyInProgress {
		fmt.Printf("❌ 项目 '%s' 不在进行中（当前状态: %s）\n", proj.Name, proj.Status)
		return
	}

	if proj.CurrentStep >= len(proj.Steps) {
		proj.Status = domain.AssemblyCompleted
		proj.CompletedAt = time.Now()
		fmt.Printf("✅ 项目 '%s' 组装完成！\n", proj.Name)
		return
	}

	step := &proj.Steps[proj.CurrentStep]

	fmt.Printf("🔨 执行步骤 %d: %s\n", step.ID, step.Description)

	// 标记为完成（简化版，实际应等待用户发射并对接）
	step.Launched = true
	step.Docked = true
	proj.CurrentStep++

	if proj.CurrentStep >= len(proj.Steps) {
		proj.Status = domain.AssemblyCompleted
		proj.CompletedAt = time.Now()
		fmt.Printf("✅ 项目 '%s' 组装完成！\n", proj.Name)
	} else {
		fmt.Printf("   下一步: %s\n", proj.Steps[proj.CurrentStep].Description)
	}
}

// assemblyPause 暂停组装项目
func (r *Repl) assemblyPause(projID string) {
	var proj *domain.AssemblyProject
	for i := range r.game.AssemblyProjects {
		if r.game.AssemblyProjects[i].ID == projID {
			proj = &r.game.AssemblyProjects[i]
			break
		}
	}
	if proj == nil {
		fmt.Printf("❌ 未找到项目 '%s'\n", projID)
		return
	}

	if proj.Status != domain.AssemblyInProgress {
		fmt.Printf("❌ 项目 '%s' 不在进行中（当前状态: %s）\n", proj.Name, proj.Status)
		return
	}

	proj.Status = domain.AssemblyPaused
	fmt.Printf("⏸️ 项目 '%s' 已暂停\n", proj.Name)
}

// assemblyResume 恢复组装项目
func (r *Repl) assemblyResume(projID string) {
	var proj *domain.AssemblyProject
	for i := range r.game.AssemblyProjects {
		if r.game.AssemblyProjects[i].ID == projID {
			proj = &r.game.AssemblyProjects[i]
			break
		}
	}
	if proj == nil {
		fmt.Printf("❌ 未找到项目 '%s'\n", projID)
		return
	}

	if proj.Status != domain.AssemblyPaused {
		fmt.Printf("❌ 项目 '%s' 未暂停（当前状态: %s）\n", proj.Name, proj.Status)
		return
	}

	proj.Status = domain.AssemblyInProgress
	fmt.Printf("▶️ 项目 '%s' 已恢复\n", proj.Name)
}

// assemblyDelete 删除组装项目
func (r *Repl) assemblyDelete(projID string) {
	var idx int = -1
	for i, p := range r.game.AssemblyProjects {
		if p.ID == projID {
			idx = i
			break
		}
	}
	if idx == -1 {
		fmt.Printf("❌ 未找到项目 '%s'\n", projID)
		return
	}

	if r.game.AssemblyProjects[idx].Status == domain.AssemblyInProgress {
		fmt.Printf("❌ 项目 '%s' 正在执行中，无法删除\n", r.game.AssemblyProjects[idx].Name)
		return
	}

	r.game.AssemblyProjects = append(r.game.AssemblyProjects[:idx], r.game.AssemblyProjects[idx+1:]...)
	fmt.Printf("✅ 项目已删除\n")
}

// assemblyProgram 关联程序到组装项目
func (r *Repl) assemblyProgram(projID, progID string) {
	var proj *domain.AssemblyProject
	for i := range r.game.AssemblyProjects {
		if r.game.AssemblyProjects[i].ID == projID {
			proj = &r.game.AssemblyProjects[i]
			break
		}
	}
	if proj == nil {
		fmt.Printf("❌ 未找到项目 '%s'\n", projID)
		return
	}

	// 验证程序是否存在
	found := false
	for _, v := range r.game.Vessels {
		for _, p := range v.ECCLPrograms {
			if p.ID == progID {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		// 检查待上传程序
		if _, ok := r.pendingPrograms[progID]; ok {
			found = true
		}
	}
	if !found {
		fmt.Printf("❌ 未找到程序 '%s'\n", progID)
		return
	}

	proj.ProgramID = progID
	fmt.Printf("✅ 程序 '%s' 已关联到项目 '%s'\n", progID, proj.Name)
}

// ==================== assembly 命令分发 ====================

func (r *Repl) handleAssemblyCommand(subCmd string, args []string) {
	switch subCmd {
	case "create":
		if len(args) < 1 {
			fmt.Println("用法: assembly create <项目名称>")
			return
		}
		r.createAssembly(args[0])
	case "add":
		if len(args) < 3 {
			fmt.Println("用法: assembly add <项目ID> <模块ID> <数量>")
			fmt.Println("  模块ID: 火箭设计ID、卫星设计ID或零件ID")
			return
		}
		r.addAssemblyStep(args[0], args[1], args[2])
	case "list":
		r.listAssemblyProjects()
	case "status":
		if len(args) < 1 {
			fmt.Println("用法: assembly status <项目ID>")
			return
		}
		r.assemblyStatus(args[0])
	case "start":
		if len(args) < 1 {
			fmt.Println("用法: assembly start <项目ID>")
			return
		}
		r.assemblyStart(args[0])
	case "step":
		if len(args) < 1 {
			fmt.Println("用法: assembly step <项目ID>")
			return
		}
		r.assemblyStep(args[0])
	case "pause":
		if len(args) < 1 {
			fmt.Println("用法: assembly pause <项目ID>")
			return
		}
		r.assemblyPause(args[0])
	case "resume":
		if len(args) < 1 {
			fmt.Println("用法: assembly resume <项目ID>")
			return
		}
		r.assemblyResume(args[0])
	case "delete":
		if len(args) < 1 {
			fmt.Println("用法: assembly delete <项目ID>")
			return
		}
		r.assemblyDelete(args[0])
	case "program":
		if len(args) < 2 {
			fmt.Println("用法: assembly program <项目ID> <程序ID>")
			return
		}
		r.assemblyProgram(args[0], args[1])
	default:
		fmt.Println("未知 assembly 子命令")
		fmt.Println("  assembly create <名称>      - 创建组装项目")
		fmt.Println("  assembly add <项目ID> <模块ID> <数量> - 添加步骤")
		fmt.Println("  assembly list              - 列出所有项目")
		fmt.Println("  assembly status <项目ID>   - 查看项目状态")
		fmt.Println("  assembly start <项目ID>    - 开始组装")
		fmt.Println("  assembly step <项目ID>     - 推进到下一步")
		fmt.Println("  assembly pause <项目ID>    - 暂停组装")
		fmt.Println("  assembly resume <项目ID>   - 恢复组装")
		fmt.Println("  assembly delete <项目ID>   - 删除项目")
		fmt.Println("  assembly program <项目ID> <程序ID> - 关联ECCL程序")
	}
}
