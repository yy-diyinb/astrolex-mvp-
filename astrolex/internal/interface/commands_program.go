package repl

import (
	"fmt"
	"strings"
	"time"

	"astrolex/internal/domain"
	"astrolex/internal/engine"
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
				Status:     "待上传",
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
			status := "已停止"
			if p.IsRunning {
				status = "运行中"
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
		fmt.Println("没有找到任何 ECCL 程序。请使用 'program edit <名称>' 创建程序。")
		return
	}
	fmt.Println("ECCL 程序列表:")
	for _, p := range progs {
		fmt.Printf("  %s: %s (状态: %s", p.ID, p.Name, p.Status)
		if p.Status != "待上传" {
			fmt.Printf(", 航天器: %s, 上传: %s", p.VesselName, p.Uploaded.Format("2006-01-02 15:04"))
		}
		fmt.Println(")")
	}
}

func (r *Repl) programEdit(progName string) {
	if progName == "" {
		fmt.Println("用法: program edit <程序名称>")
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
		fmt.Printf("编辑程序 '%s' (航天器: %s)\n", progName, existingVessel.Name)
		fmt.Printf("当前代码:\n%s\n", existingProg.Code)
		fmt.Println("请输入新代码（输入 'EOF' 单独一行结束）：")
	} else {
		fmt.Printf("创建新程序 '%s'\n", progName)
		fmt.Println("请输入 ECCL 代码（输入 'EOF' 单独一行结束）：")
		fmt.Println("示例:\n  :START\n  SET COUNTER 0\n  :LOOP\n  LOG \"Hello from ECCL\"\n  WAIT 10\n  SET COUNTER $COUNTER + 1\n  IF $COUNTER > 5 GOTO :END\n  GOTO :LOOP\n  :END\n  LOG \"程序结束\"")
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
		fmt.Println("程序代码为空，取消操作。")
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
		fmt.Printf("程序解析错误: %v\n", err)
		return
	}

	if existingProg != nil {
		existingProg.Code = code
		existingProg.Registers = prog.Registers
		existingProg.Labels = prog.Labels
		existingProg.UploadedAt = time.Now()
		fmt.Printf("程序 '%s' 已更新\n", progName)
	} else {
		if r.pendingPrograms == nil {
			r.pendingPrograms = make(map[string]*domain.ECCLProgram)
		}
		prog.ID = fmt.Sprintf("prog_%d", len(r.pendingPrograms)+1)
		r.pendingPrograms[prog.ID] = prog
		fmt.Printf("程序 '%s' 已创建，ID: %s\n", progName, prog.ID)
		fmt.Println("使用 'program upload <程序ID> <航天器ID>' 上传到航天器")
	}
}

func (r *Repl) programUpload(progID, vesselID string) {
	if progID == "" || vesselID == "" {
		fmt.Println("用法: program upload <程序ID> <航天器ID>")
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
					fmt.Printf("程序 '%s' 已存在于航天器 '%s' 上\n", prog.Name, v.Name)
					return
				}
			}
		}
		fmt.Printf("错误: 未找到程序 ID '%s'\n", progID)
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
		fmt.Printf("错误: 未找到航天器 ID '%s'\n", vesselID)
		return
	}

	if !vessel.HasControlCenter {
		fmt.Println("⚠️ 警告: 航天器没有控制中心，程序可以上传但无法运行")
		fmt.Println("   请确保航天器包含航电 (av-1) 才能执行程序")
	}

	prog.UploadedAt = time.Now()
	prog.IsRunning = false
	vessel.ECCLPrograms = append(vessel.ECCLPrograms, *prog)
	delete(r.pendingPrograms, progID)

	fmt.Printf("程序 '%s' 已上传到航天器 '%s'\n", prog.Name, vessel.Name)
	fmt.Println("使用 'program run <程序ID> <航天器ID>' 执行程序")
}

func (r *Repl) programRun(progID, vesselID string) {
	if progID == "" || vesselID == "" {
		fmt.Println("用法: program run <程序ID> <航天器ID>")
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
		fmt.Printf("错误: 未找到航天器 ID '%s'\n", vesselID)
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
		fmt.Printf("错误: 航天器上未找到程序 ID '%s'\n", progID)
		return
	}

	if prog.IsRunning {
		fmt.Printf("程序 '%s' 已在运行中\n", prog.Name)
		return
	}

	if !vessel.HasControlCenter {
		fmt.Println("❌ 错误: 航天器没有控制中心，无法执行程序")
		fmt.Println("   请确保航天器包含航电 (av-1)")
		return
	}

	vm := engine.NewECCLVM(vessel, prog)
	vm.IsRunning = true
	prog.IsRunning = true
	prog.LastExecuted = time.Now()

	fmt.Printf("程序 '%s' 开始在航天器 '%s' 上执行...\n", prog.Name, vessel.Name)
	fmt.Println("(使用 'program stop <程序ID> <航天器ID>' 停止执行)")

	go func() {
		err := vm.Run()
		if err != nil {
			fmt.Printf("程序执行错误: %v\n", err)
		}
		prog.IsRunning = false
		if len(vm.LastLog) > 0 {
			prog.Logs = append(prog.Logs, vm.LastLog...)
			if len(prog.Logs) > 100 {
				prog.Logs = prog.Logs[len(prog.Logs)-100:]
			}
		}
		fmt.Printf("程序 '%s' 执行完成\n", prog.Name)
	}()
}

func (r *Repl) programStop(progID, vesselID string) {
	if progID == "" || vesselID == "" {
		fmt.Println("用法: program stop <程序ID> <航天器ID>")
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
		fmt.Printf("错误: 未找到航天器 ID '%s'\n", vesselID)
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
		fmt.Printf("错误: 航天器上未找到程序 ID '%s'\n", progID)
		return
	}

	if !prog.IsRunning {
		fmt.Printf("程序 '%s' 未在运行\n", prog.Name)
		return
	}

	prog.IsRunning = false
	fmt.Printf("程序 '%s' 已停止\n", prog.Name)
}

func (r *Repl) programLogs(progID, vesselID string) {
	if progID == "" || vesselID == "" {
		fmt.Println("用法: program logs <程序ID> <航天器ID>")
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
		fmt.Printf("错误: 未找到航天器 ID '%s'\n", vesselID)
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
		fmt.Printf("错误: 航天器上未找到程序 ID '%s'\n", progID)
		return
	}

	if len(prog.Logs) == 0 {
		fmt.Printf("程序 '%s' 没有日志\n", prog.Name)
		return
	}
	fmt.Printf("程序 '%s' 日志 (最新 20 条):\n", prog.Name)
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
		fmt.Println("用法: program status <程序ID> <航天器ID>")
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
		fmt.Printf("错误: 未找到航天器 ID '%s'\n", vesselID)
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
		fmt.Printf("错误: 航天器上未找到程序 ID '%s'\n", progID)
		return
	}

	status := "停止"
	if prog.IsRunning {
		status = "运行中"
	}
	fmt.Printf("=== 程序 '%s' 状态 ===\n", prog.Name)
	fmt.Printf("ID: %s\n", prog.ID)
	fmt.Printf("状态: %s\n", status)
	fmt.Printf("航天器: %s (%s)\n", vessel.Name, vessel.ID)
	fmt.Printf("上传时间: %s\n", prog.UploadedAt.Format("2006-01-02 15:04:05"))
	if !prog.LastExecuted.IsZero() {
		fmt.Printf("最后执行: %s\n", prog.LastExecuted.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("寄存器:\n")
	for k, v := range prog.Registers {
		fmt.Printf("  $%s = %.2f\n", k, v)
	}
	fmt.Printf("标签数量: %d\n", len(prog.Labels))
	fmt.Printf("日志条数: %d\n", len(prog.Logs))
	if vessel.HasControlCenter {
		fmt.Println("控制中心: ✅ 是")
	} else {
		fmt.Println("控制中心: ❌ 否 (程序可以上传但无法运行)")
	}
}

func (r *Repl) handleProgramCommand(subCmd string, args []string) {
	switch subCmd {
	case "list":
		r.programList()
	case "edit":
		if len(args) < 1 {
			fmt.Println("用法: program edit <程序名称>")
			return
		}
		r.programEdit(args[0])
	case "upload":
		if len(args) < 2 {
			fmt.Println("用法: program upload <程序ID> <航天器ID>")
			return
		}
		r.programUpload(args[0], args[1])
	case "run":
		if len(args) < 2 {
			fmt.Println("用法: program run <程序ID> <航天器ID>")
			return
		}
		r.programRun(args[0], args[1])
	case "stop":
		if len(args) < 2 {
			fmt.Println("用法: program stop <程序ID> <航天器ID>")
			return
		}
		r.programStop(args[0], args[1])
	case "logs":
		if len(args) < 2 {
			fmt.Println("用法: program logs <程序ID> <航天器ID>")
			return
		}
		r.programLogs(args[0], args[1])
	case "status":
		if len(args) < 2 {
			fmt.Println("用法: program status <程序ID> <航天器ID>")
			return
		}
		r.programStatus(args[0], args[1])
	default:
		fmt.Println("未知 program 子命令")
		fmt.Println("  program list                    - 列出所有程序（含待上传）")
		fmt.Println("  program edit <名称>             - 创建或编辑程序")
		fmt.Println("  program upload <程序ID> <航天器ID> - 上传程序到航天器")
		fmt.Println("  program run <程序ID> <航天器ID>   - 在航天器上运行程序")
		fmt.Println("  program stop <程序ID> <航天器ID>  - 停止运行中的程序")
		fmt.Println("  program logs <程序ID> <航天器ID>  - 查看程序日志")
		fmt.Println("  program status <程序ID> <航天器ID> - 查看程序状态")
	}
}
