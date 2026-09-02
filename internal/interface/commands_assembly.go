package repl

import (
	"fmt"
	"strconv"
	"time"

	"astrolex/internal/domain"
	"astrolex/internal/i18n"
)

// ==================== 组装项目管理命令 ====================

// createAssembly 创建组装项目
func (r *Repl) createAssembly(name string) {
	if name == "" {
		fmt.Println(i18n.T("assembly_create_usage"))
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
	fmt.Printf(i18n.T("assembly_create_success"), name, proj.ID)
	fmt.Println(i18n.T("assembly_create_hint"))
}

// addAssemblyStep 添加组装步骤
func (r *Repl) addAssemblyStep(projID, moduleID, quantityStr string) {
	qty, err := strconv.Atoi(quantityStr)
	if err != nil || qty <= 0 {
		fmt.Println(i18n.T("assembly_add_invalid_quantity"))
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
		fmt.Printf(i18n.T("assembly_add_not_found"), projID)
		return
	}

	if proj.Status == domain.AssemblyCompleted {
		fmt.Printf(i18n.T("assembly_add_completed"), proj.Name)
		return
	}

	// 验证模块ID是否存在（火箭设计、卫星设计或零件）
	exists := false
	moduleType := i18n.T("assembly_module_unknown")

	for _, d := range r.game.RocketDesigns {
		if d.ID == moduleID {
			exists = true
			moduleType = i18n.T("assembly_module_rocket")
			break
		}
	}
	if !exists {
		for _, d := range r.game.SatelliteDesigns {
			if d.ID == moduleID {
				exists = true
				moduleType = i18n.T("assembly_module_satellite")
				break
			}
		}
	}
	if !exists {
		if _, ok := r.game.PartsDB[moduleID]; ok {
			exists = true
			moduleType = i18n.T("assembly_module_part")
		}
	}
	if !exists {
		fmt.Printf(i18n.T("assembly_add_module_not_found"), moduleID)
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
	fmt.Printf(i18n.T("assembly_add_success"), step.ID, step.Description)
}

// listAssemblyProjects 列出所有组装项目
func (r *Repl) listAssemblyProjects() {
	if len(r.game.AssemblyProjects) == 0 {
		fmt.Println(i18n.T("assembly_list_no_projects"))
		return
	}
	fmt.Println(i18n.T("assembly_list_title"))
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
		statusStr := string(p.Status)
		fmt.Printf(i18n.T("assembly_list_item"),
			statusIcon, p.ID, p.Name, statusStr, progress, p.CurrentStep, len(p.Steps), p.CreatedAt.Format("2006-01-02"))
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
		fmt.Printf(i18n.T("assembly_status_not_found"), projID)
		return
	}

	fmt.Printf(i18n.T("assembly_status_title"), proj.Name, proj.ID)
	fmt.Printf(i18n.T("assembly_status_state"), proj.Status)
	fmt.Printf(i18n.T("assembly_status_steps"), proj.CurrentStep, len(proj.Steps))
	fmt.Printf(i18n.T("assembly_status_created"), proj.CreatedAt.Format("2006-01-02 15:04:05"))
	if !proj.StartedAt.IsZero() {
		fmt.Printf(i18n.T("assembly_status_started"), proj.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if !proj.CompletedAt.IsZero() {
		fmt.Printf(i18n.T("assembly_status_completed"), proj.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	if proj.ProgramID != "" {
		fmt.Printf(i18n.T("assembly_status_program"), proj.ProgramID)
	}

	fmt.Println(i18n.T("assembly_status_steps_list"))
	if len(proj.Steps) == 0 {
		fmt.Println(i18n.T("assembly_status_no_steps"))
		return
	}
	for i, step := range proj.Steps {
		statusIcon := "⏳"
		if i < proj.CurrentStep {
			statusIcon = "✅"
		} else if i == proj.CurrentStep {
			statusIcon = "🔨"
		}
		statusText := i18n.T("assembly_status_waiting")
		if i < proj.CurrentStep {
			statusText = i18n.T("assembly_status_done")
		} else if i == proj.CurrentStep {
			statusText = i18n.T("assembly_status_in_progress")
		}
		launchStatus := i18n.T("assembly_status_not_launched")
		if step.Launched {
			launchStatus = i18n.T("assembly_status_launched")
		}
		dockStatus := i18n.T("assembly_status_not_docked")
		if step.Docked {
			dockStatus = i18n.T("assembly_status_docked")
		}
		fmt.Printf(i18n.T("assembly_status_step"),
			statusIcon, step.ID, step.Description, statusText, launchStatus, dockStatus)
		if step.VesselID != "" {
			fmt.Printf(i18n.T("assembly_status_vessel"), step.VesselID)
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
		fmt.Printf(i18n.T("assembly_start_not_found"), projID)
		return
	}

	if proj.Status == domain.AssemblyCompleted {
		fmt.Printf(i18n.T("assembly_start_completed"), proj.Name)
		return
	}

	if proj.Status == domain.AssemblyInProgress {
		fmt.Printf(i18n.T("assembly_start_in_progress"), proj.Name)
		return
	}

	if len(proj.Steps) == 0 {
		fmt.Printf(i18n.T("assembly_start_no_steps"), proj.Name)
		return
	}

	proj.Status = domain.AssemblyInProgress
	proj.StartedAt = time.Now()
	proj.CurrentStep = 0

	fmt.Printf(i18n.T("assembly_start_success"), proj.Name)
	fmt.Println(i18n.T("assembly_start_hint"))
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
		fmt.Printf(i18n.T("assembly_step_not_found"), projID)
		return
	}

	if proj.Status != domain.AssemblyInProgress {
		fmt.Printf(i18n.T("assembly_step_not_in_progress"), proj.Name, proj.Status)
		return
	}

	if proj.CurrentStep >= len(proj.Steps) {
		proj.Status = domain.AssemblyCompleted
		proj.CompletedAt = time.Now()
		fmt.Printf(i18n.T("assembly_step_completed"), proj.Name)
		return
	}

	step := &proj.Steps[proj.CurrentStep]

	fmt.Printf(i18n.T("assembly_step_executing"), step.ID, step.Description)

	// 标记为完成（简化版，实际应等待用户发射并对接）
	step.Launched = true
	step.Docked = true
	proj.CurrentStep++

	if proj.CurrentStep >= len(proj.Steps) {
		proj.Status = domain.AssemblyCompleted
		proj.CompletedAt = time.Now()
		fmt.Printf(i18n.T("assembly_step_completed"), proj.Name)
	} else {
		fmt.Printf(i18n.T("assembly_step_next"), proj.Steps[proj.CurrentStep].Description)
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
		fmt.Printf(i18n.T("assembly_pause_not_found"), projID)
		return
	}

	if proj.Status != domain.AssemblyInProgress {
		fmt.Printf(i18n.T("assembly_pause_not_in_progress"), proj.Name, proj.Status)
		return
	}

	proj.Status = domain.AssemblyPaused
	fmt.Printf(i18n.T("assembly_pause_success"), proj.Name)
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
		fmt.Printf(i18n.T("assembly_resume_not_found"), projID)
		return
	}

	if proj.Status != domain.AssemblyPaused {
		fmt.Printf(i18n.T("assembly_resume_not_paused"), proj.Name, proj.Status)
		return
	}

	proj.Status = domain.AssemblyInProgress
	fmt.Printf(i18n.T("assembly_resume_success"), proj.Name)
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
		fmt.Printf(i18n.T("assembly_delete_not_found"), projID)
		return
	}

	if r.game.AssemblyProjects[idx].Status == domain.AssemblyInProgress {
		fmt.Printf(i18n.T("assembly_delete_in_progress"), r.game.AssemblyProjects[idx].Name)
		return
	}

	r.game.AssemblyProjects = append(r.game.AssemblyProjects[:idx], r.game.AssemblyProjects[idx+1:]...)
	fmt.Println(i18n.T("assembly_delete_success"))
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
		fmt.Printf(i18n.T("assembly_program_not_found"), projID)
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
		fmt.Printf(i18n.T("assembly_program_prog_not_found"), progID)
		return
	}

	proj.ProgramID = progID
	fmt.Printf(i18n.T("assembly_program_success"), progID, proj.Name)
}

// ==================== assembly 命令分发 ====================

func (r *Repl) handleAssemblyCommand(subCmd string, args []string) {
	switch subCmd {
	case "create":
		if len(args) < 1 {
			fmt.Println(i18n.T("assembly_create_usage"))
			return
		}
		r.createAssembly(args[0])
	case "add":
		if len(args) < 3 {
			fmt.Println(i18n.T("assembly_add_usage"))
			return
		}
		r.addAssemblyStep(args[0], args[1], args[2])
	case "list":
		r.listAssemblyProjects()
	case "status":
		if len(args) < 1 {
			fmt.Println(i18n.T("assembly_status_usage"))
			return
		}
		r.assemblyStatus(args[0])
	case "start":
		if len(args) < 1 {
			fmt.Println(i18n.T("assembly_start_usage"))
			return
		}
		r.assemblyStart(args[0])
	case "step":
		if len(args) < 1 {
			fmt.Println(i18n.T("assembly_step_usage"))
			return
		}
		r.assemblyStep(args[0])
	case "pause":
		if len(args) < 1 {
			fmt.Println(i18n.T("assembly_pause_usage"))
			return
		}
		r.assemblyPause(args[0])
	case "resume":
		if len(args) < 1 {
			fmt.Println(i18n.T("assembly_resume_usage"))
			return
		}
		r.assemblyResume(args[0])
	case "delete":
		if len(args) < 1 {
			fmt.Println(i18n.T("assembly_delete_usage"))
			return
		}
		r.assemblyDelete(args[0])
	case "program":
		if len(args) < 2 {
			fmt.Println(i18n.T("assembly_program_usage"))
			return
		}
		r.assemblyProgram(args[0], args[1])
	default:
		fmt.Println(i18n.T("assembly_unknown_command"))
		fmt.Println(i18n.T("assembly_help"))
		fmt.Println(i18n.T("assembly_help_create"))
		fmt.Println(i18n.T("assembly_help_add"))
		fmt.Println(i18n.T("assembly_help_list"))
		fmt.Println(i18n.T("assembly_help_status"))
		fmt.Println(i18n.T("assembly_help_start"))
		fmt.Println(i18n.T("assembly_help_step"))
		fmt.Println(i18n.T("assembly_help_pause"))
		fmt.Println(i18n.T("assembly_help_resume"))
		fmt.Println(i18n.T("assembly_help_delete"))
		fmt.Println(i18n.T("assembly_help_program"))
	}
}
