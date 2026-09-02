package repl

import (
	"fmt"
	"strings"
	"time"

	"astrolex/internal/domain"
	"astrolex/internal/engine"
	"astrolex/internal/i18n"
)

// ==================== 程序管理命令 ====================

func (r *Repl) programList() {
	type progInfo struct {
		ID         string
		Name       string
		Status     string   // "待上传", "已上传", "运行中", "已停止"
		VesselName string
		VesselID   string
		Uploaded   time.Time
	}
	var progs []progInfo

	// 1. 待上传的程序
	if r.pendingPrograms != nil {
		for _, p := range r.pendingPrograms {
			progs = append(progs, progInfo{
				ID:         p.ID,
				Name:       p.Name,
				Status:     i18n.T("program_status_pending"),
				VesselName: "-",
				VesselID:   "-",
				Uploaded:   time.Time{},
			})
		}
	}

	// 2. 已上传的程序
	for _, v := range r.game.Vessels {
		if !v.IsActive {
			continue
		}
		for _, p := range v.ECCLPrograms {
			status := i18n.T("program_status_stopped")
			if p.IsRunning {
				status = i18n.T("program_status_running")
			}
			progs = append(progs, progInfo{
				ID:         p.ID,
				Name:       p.Name,
				Status:     status,
				VesselName: v.Name,
				VesselID:   v.ID,
				Uploaded:   p.UploadedAt,
			})
		}
	}

	if len(progs) == 0 {
		fmt.Println(i18n.T("program_list_none"))
		return
	}
	fmt.Println(i18n.T("program_list_title"))
	for _, p := range progs {
		if p.Status == i18n.T("program_status_pending") {
			fmt.Printf(i18n.T("program_list_pending"), p.ID, p.Name)
		} else {
			fmt.Printf(i18n.T("program_list_item"),
				p.ID, p.Name, p.Status, p.VesselName, p.Uploaded.Format("2006-01-02 15:04"))
		}
	}
}

func (r *Repl) programEdit(progName string) {
	if progName == "" {
		fmt.Println(i18n.T("program_edit_usage"))
		return
	}

	var existingProg *domain.ECCLProgram
	var existingVessel *domain.Vessel
	for _, v := range r.game.Vessels {
		if !v.IsActive {
			continue
		}
		for i := range v.ECCLPrograms {
			if v.ECCLPrograms[i].Name == progName {
				existingProg = &v.ECCLPrograms[i]
				existingVessel = &v
				break
			}
		}
		if existingProg != nil {
			break
		}
	}

	var code string
	if existingProg != nil {
		fmt.Printf(i18n.T("program_edit_edit"), progName, existingVessel.Name)
		fmt.Printf(i18n.T("program_edit_current_code"), existingProg.Code)
		fmt.Println(i18n.T("program_edit_enter_code"))
	} else {
		fmt.Printf(i18n.T("program_edit_create"), progName)
		fmt.Println(i18n.T("program_edit_enter_code"))
		fmt.Println(i18n.T("program_edit_example"))
	}

	var lines []string
	for {
		fmt.Print("> ")
		line, _ := r.reader.ReadString('\n')
		line = strings.TrimRight(line, "\n")
		if line == "EOF" {
			break
		}
		lines = append(lines, line)
	}
	code = strings.Join(lines, "\n")

	if len(strings.TrimSpace(code)) == 0 {
		fmt.Println(i18n.T("program_edit_empty"))
		return
	}

	prog := &domain.ECCLProgram{
		Name:        progName,
		Code:        code,
		Registers:   make(map[string]float64),
		Labels:      make(map[string]int),
		UploadedAt:  time.Now(),
		IsRunning:   false,
		Logs:        []string{},
	}

	if err := engine.LoadProgram(prog); err != nil {
		fmt.Printf(i18n.T("program_edit_parse_error"), err)
		return
	}

	if existingProg != nil {
		existingProg.Code = code
		existingProg.Registers = prog.Registers
		existingProg.Labels = prog.Labels
		existingProg.UploadedAt = time.Now()
		fmt.Printf(i18n.T("program_edit_updated"), progName)
	} else {
		if r.pendingPrograms == nil {
			r.pendingPrograms = make(map[string]*domain.ECCLProgram)
		}
		prog.ID = fmt.Sprintf("prog_%d", len(r.pendingPrograms)+1)
		r.pendingPrograms[prog.ID] = prog
		fmt.Printf(i18n.T("program_edit_created"), progName, prog.ID)
		fmt.Println(i18n.T("program_edit_upload_hint"))
	}
}

func (r *Repl) programUpload(progID, vesselID string) {
	if progID == "" || vesselID == "" {
		fmt.Println(i18n.T("program_upload_usage"))
		return
	}

	var prog *domain.ECCLProgram
	if r.pendingPrograms != nil {
		if p, ok := r.pendingPrograms[progID]; ok {
			prog = p
		}
	}
	if prog == nil {
		for _, v := range r.game.Vessels {
			if !v.IsActive {
				continue
			}
			for i := range v.ECCLPrograms {
				if v.ECCLPrograms[i].ID == progID {
					fmt.Printf(i18n.T("program_upload_already_exists"), prog.Name, v.Name)
					return
				}
			}
		}
		fmt.Printf(i18n.T("program_upload_not_found"), progID)
		return
	}

	var vessel *domain.Vessel
	for i := range r.game.Vessels {
		if r.game.Vessels[i].ID == vesselID && r.game.Vessels[i].IsActive {
			vessel = &r.game.Vessels[i]
			break
		}
	}
	if vessel == nil {
		fmt.Printf(i18n.T("program_upload_vessel_not_found"), vesselID)
		return
	}

	if !vessel.HasControlCenter {
		fmt.Println(i18n.T("program_upload_warning_no_control"))
	}

	prog.UploadedAt = time.Now()
	prog.IsRunning = false
	vessel.ECCLPrograms = append(vessel.ECCLPrograms, *prog)
	delete(r.pendingPrograms, progID)

	fmt.Printf(i18n.T("program_upload_success"), prog.Name, vessel.Name)
	fmt.Println(i18n.T("program_upload_run_hint"))
}

func (r *Repl) programRun(progID, vesselID string) {
	if progID == "" || vesselID == "" {
		fmt.Println(i18n.T("program_run_usage"))
		return
	}

	var vessel *domain.Vessel
	for i := range r.game.Vessels {
		if r.game.Vessels[i].ID == vesselID && r.game.Vessels[i].IsActive {
			vessel = &r.game.Vessels[i]
			break
		}
	}
	if vessel == nil {
		fmt.Printf(i18n.T("program_run_vessel_not_found"), vesselID)
		return
	}

	var prog *domain.ECCLProgram
	for i := range vessel.ECCLPrograms {
		if vessel.ECCLPrograms[i].ID == progID {
			prog = &vessel.ECCLPrograms[i]
			break
		}
	}
	if prog == nil {
		fmt.Printf(i18n.T("program_run_not_found"), progID)
		return
	}

	if prog.IsRunning {
		fmt.Printf(i18n.T("program_run_already_running"), prog.Name)
		return
	}

	if !vessel.HasControlCenter {
		fmt.Println(i18n.T("program_run_no_control"))
		return
	}

	vm := engine.NewECCLVM(vessel, prog)
	vm.IsRunning = true
	prog.IsRunning = true
	prog.LastExecuted = time.Now()

	fmt.Printf(i18n.T("program_run_start"), prog.Name, vessel.Name)
	fmt.Println(i18n.T("program_run_stop_hint"))

	go func() {
		err := vm.Run()
		if err != nil {
			fmt.Printf(i18n.T("program_run_error"), err)
		}
		prog.IsRunning = false
		if len(vm.LastLog) > 0 {
			prog.Logs = append(prog.Logs, vm.LastLog...)
			if len(prog.Logs) > 100 {
				prog.Logs = prog.Logs[len(prog.Logs)-100:]
			}
		}
		fmt.Printf(i18n.T("program_run_complete"), prog.Name)
	}()
}

func (r *Repl) programStop(progID, vesselID string) {
	if progID == "" || vesselID == "" {
		fmt.Println(i18n.T("program_stop_usage"))
		return
	}

	var vessel *domain.Vessel
	for i := range r.game.Vessels {
		if r.game.Vessels[i].ID == vesselID && r.game.Vessels[i].IsActive {
			vessel = &r.game.Vessels[i]
			break
		}
	}
	if vessel == nil {
		fmt.Printf(i18n.T("program_stop_vessel_not_found"), vesselID)
		return
	}

	var prog *domain.ECCLProgram
	for i := range vessel.ECCLPrograms {
		if vessel.ECCLPrograms[i].ID == progID {
			prog = &vessel.ECCLPrograms[i]
			break
		}
	}
	if prog == nil {
		fmt.Printf(i18n.T("program_stop_not_found"), progID)
		return
	}

	if !prog.IsRunning {
		fmt.Printf(i18n.T("program_stop_not_running"), prog.Name)
		return
	}

	prog.IsRunning = false
	fmt.Printf(i18n.T("program_stop_success"), prog.Name)
}

func (r *Repl) programLogs(progID, vesselID string) {
	if progID == "" || vesselID == "" {
		fmt.Println(i18n.T("program_logs_usage"))
		return
	}

	var vessel *domain.Vessel
	for i := range r.game.Vessels {
		if r.game.Vessels[i].ID == vesselID && r.game.Vessels[i].IsActive {
			vessel = &r.game.Vessels[i]
			break
		}
	}
	if vessel == nil {
		fmt.Printf(i18n.T("program_logs_vessel_not_found"), vesselID)
		return
	}

	var prog *domain.ECCLProgram
	for i := range vessel.ECCLPrograms {
		if vessel.ECCLPrograms[i].ID == progID {
			prog = &vessel.ECCLPrograms[i]
			break
		}
	}
	if prog == nil {
		fmt.Printf(i18n.T("program_logs_not_found"), progID)
		return
	}

	if len(prog.Logs) == 0 {
		fmt.Printf(i18n.T("program_logs_no_logs"), prog.Name)
		return
	}
	fmt.Printf(i18n.T("program_logs_title"), prog.Name)
	start := 0
	if len(prog.Logs) > 20 {
		start = len(prog.Logs) - 20
	}
	for _, log := range prog.Logs[start:] {
		fmt.Println("  " + log)
	}
}

func (r *Repl) programStatus(progID, vesselID string) {
	if progID == "" || vesselID == "" {
		fmt.Println(i18n.T("program_status_usage"))
		return
	}

	var vessel *domain.Vessel
	for i := range r.game.Vessels {
		if r.game.Vessels[i].ID == vesselID && r.game.Vessels[i].IsActive {
			vessel = &r.game.Vessels[i]
			break
		}
	}
	if vessel == nil {
		fmt.Printf(i18n.T("program_status_vessel_not_found"), vesselID)
		return
	}

	var prog *domain.ECCLProgram
	for i := range vessel.ECCLPrograms {
		if vessel.ECCLPrograms[i].ID == progID {
			prog = &vessel.ECCLPrograms[i]
			break
		}
	}
	if prog == nil {
		fmt.Printf(i18n.T("program_status_not_found"), progID)
		return
	}

	status := i18n.T("program_status_stopped")
	if prog.IsRunning {
		status = i18n.T("program_status_running")
	}
	fmt.Printf(i18n.T("program_status_title"), prog.Name)
	fmt.Printf(i18n.T("program_status_id"), prog.ID)
	fmt.Printf(i18n.T("program_status_state"), status)
	fmt.Printf(i18n.T("program_status_vessel"), vessel.Name, vessel.ID)
	fmt.Printf(i18n.T("program_status_uploaded"), prog.UploadedAt.Format("2006-01-02 15:04:05"))
	if !prog.LastExecuted.IsZero() {
		fmt.Printf(i18n.T("program_status_last_executed"), prog.LastExecuted.Format("2006-01-02 15:04:05"))
	}
	fmt.Println(i18n.T("program_status_registers"))
	for k, v := range prog.Registers {
		fmt.Printf(i18n.T("program_status_register"), k, v)
	}
	fmt.Printf(i18n.T("program_status_labels"), len(prog.Labels))
	fmt.Printf(i18n.T("program_status_logs"), len(prog.Logs))
	if vessel.HasControlCenter {
		fmt.Println(i18n.T("program_status_control_yes"))
	} else {
		fmt.Println(i18n.T("program_status_control_no"))
	}
}

func (r *Repl) handleProgramCommand(subCmd string, args []string) {
	switch subCmd {
	case "list":
		r.programList()
	case "edit":
		if len(args) < 1 {
			fmt.Println(i18n.T("program_edit_usage"))
			return
		}
		r.programEdit(args[0])
	case "upload":
		if len(args) < 2 {
			fmt.Println(i18n.T("program_upload_usage"))
			return
		}
		r.programUpload(args[0], args[1])
	case "run":
		if len(args) < 2 {
			fmt.Println(i18n.T("program_run_usage"))
			return
		}
		r.programRun(args[0], args[1])
	case "stop":
		if len(args) < 2 {
			fmt.Println(i18n.T("program_stop_usage"))
			return
		}
		r.programStop(args[0], args[1])
	case "logs":
		if len(args) < 2 {
			fmt.Println(i18n.T("program_logs_usage"))
			return
		}
		r.programLogs(args[0], args[1])
	case "status":
		if len(args) < 2 {
			fmt.Println(i18n.T("program_status_usage"))
			return
		}
		r.programStatus(args[0], args[1])
	default:
		fmt.Println(i18n.T("program_unknown_command"))
		fmt.Println(i18n.T("program_help"))
		fmt.Println(i18n.T("program_help_list"))
		fmt.Println(i18n.T("program_help_edit"))
		fmt.Println(i18n.T("program_help_upload"))
		fmt.Println(i18n.T("program_help_run"))
		fmt.Println(i18n.T("program_help_stop"))
		fmt.Println(i18n.T("program_help_logs"))
		fmt.Println(i18n.T("program_help_status"))
	}
}
