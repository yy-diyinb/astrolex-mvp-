package engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"astrolex/internal/domain"
)

// ECCLVM 代表一个 ECCL 虚拟机实例，关联到一颗卫星
type ECCLVM struct {
	Satellite      *domain.Vessel
	Program        *domain.ECCLProgram
	InstructionPtr int
	Registers      map[string]float64
	Labels         map[string]int
	IsRunning      bool
	LastLog        []string

	// 调用堆栈（用于 CALL/RET）
	CallStack []int

	// 事件处理器（已注册的事件）
	// 不需要队列，事件通过 TriggerEvent 直接跳转
}

// 指令集常量
const (
	InstrSet      = "SET"
	InstrWait     = "WAIT"
	InstrAct      = "ACT"
	InstrOnEvent  = "ON EVENT"
	InstrLog      = "LOG"
	InstrJettison = "JETTISON"
	InstrOrient   = "ORIENT"
	InstrShutdown = "SHUTDOWN"
	InstrIf       = "IF"
	InstrGoto     = "GOTO"
	InstrLabel    = "LABEL"
	InstrMeasure  = "MEASURE"   // 采集数据
	InstrSend     = "SEND"      // 发送数据
	InstrPing     = "PING"      // 通信检测
	InstrFire     = "FIRE"      // 引擎点火
	InstrCalc     = "CALC"      // 计算表达式
	InstrCall     = "CALL"      // 调用子程序
	InstrRet      = "RET"       // 从子程序返回
)

// NewECCLVM 创建新的虚拟机实例
func NewECCLVM(vessel *domain.Vessel, prog *domain.ECCLProgram) *ECCLVM {
	vm := &ECCLVM{
		Satellite:      vessel,
		Program:        prog,
		InstructionPtr: 0,
		Registers:      make(map[string]float64),
		Labels:         prog.Labels,
		IsRunning:      false,
		LastLog:        []string{},
		CallStack:      []int{},
	}
	for k, v := range prog.Registers {
		vm.Registers[k] = v
	}
	return vm
}

// parseInstruction 解析单行指令，返回指令名和参数列表
func parseInstruction(line string) (string, []string, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
		return "", nil, nil
	}
	if strings.HasPrefix(line, ":") {
		return InstrLabel, []string{strings.TrimPrefix(line, ":")}, nil
	}
	if strings.HasPrefix(strings.ToUpper(line), "ON EVENT") {
		parts := strings.Fields(line)
		if len(parts) < 5 {
			return "", nil, errors.New("ON EVENT 格式错误")
		}
		eventName := parts[2]
		operator := parts[3]
		valueStr := parts[4]
		label := parts[5]
		if !strings.HasPrefix(label, ":") {
			label = ":" + label
		}
		return "ON EVENT", []string{eventName, operator, valueStr, label}, nil
	}
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return "", nil, nil
	}
	cmd := strings.ToUpper(parts[0])
	// 处理 LOG 指令，允许参数包含空格
	if cmd == "LOG" {
		if len(parts) > 1 {
			return cmd, []string{strings.Join(parts[1:], " ")}, nil
		}
		return cmd, []string{}, nil
	}
	// 处理 CALC 指令，参数是表达式
	if cmd == "CALC" {
		if len(parts) < 4 {
			return "", nil, errors.New("CALC 格式: CALC <结果寄存器> <表达式>")
		}
		return cmd, parts[1:], nil
	}
	return cmd, parts[1:], nil
}

// executeInstruction 执行单条指令
func (vm *ECCLVM) executeInstruction(instr string, args []string) (waitSeconds float64, err error) {
	switch instr {
	case InstrLabel:
		return 0, nil
	case InstrSet:
		if len(args) < 2 {
			return 0, errors.New("SET 需要至少两个参数")
		}
		regName := args[0]
		valueStr := args[1]
		var val float64
		if strings.HasPrefix(valueStr, "$") {
			reg := strings.TrimPrefix(valueStr, "$")
			if v, ok := vm.Registers[reg]; ok {
				val = v
			} else {
				return 0, fmt.Errorf("未找到寄存器 %s", reg)
			}
		} else {
			var err error
			val, err = strconv.ParseFloat(valueStr, 64)
			if err != nil {
				return 0, fmt.Errorf("无效数值 %s", valueStr)
			}
		}
		vm.Registers[regName] = val
		vm.Program.Registers[regName] = val
		return 0, nil
	case InstrWait:
		if len(args) < 1 {
			return 0, errors.New("WAIT 需要时间参数")
		}
		secStr := args[0]
		var sec float64
		if strings.HasPrefix(secStr, "$") {
			reg := strings.TrimPrefix(secStr, "$")
			if v, ok := vm.Registers[reg]; ok {
				sec = v
			} else {
				return 0, fmt.Errorf("未找到寄存器 %s", reg)
			}
		} else {
			var err error
			sec, err = strconv.ParseFloat(secStr, 64)
			if err != nil {
				return 0, fmt.Errorf("无效数值 %s", secStr)
			}
		}
		return sec, nil
	case InstrAct:
		if len(args) < 1 {
			return 0, errors.New("ACT 需要动作参数")
		}
		action := args[0]
		switch action {
		case "LIFE_SUPPORT_CHECK":
			vm.Log("执行生命支持检查")
		case "POWER_MONITOR":
			vm.Log(fmt.Sprintf("当前电力: %.0f Wh", vm.Satellite.Power))
		case "ASSEMBLY_STEP":
			vm.Log("执行组装步骤检查")
		default:
			vm.Log(fmt.Sprintf("执行动作: %s", action))
		}
		return 0, nil
	case InstrOnEvent:
		if len(args) < 4 {
			return 0, errors.New("ON EVENT 需要4个参数")
		}
		eventName := args[0]
		operator := args[1]
		valueStr := args[2]
		label := args[3]
		if strings.HasPrefix(label, ":") {
			label = strings.TrimPrefix(label, ":")
		}
		var val float64
		if strings.HasPrefix(valueStr, "$") {
			reg := strings.TrimPrefix(valueStr, "$")
			if v, ok := vm.Registers[reg]; ok {
				val = v
			} else {
				return 0, fmt.Errorf("未找到寄存器 %s", reg)
			}
		} else {
			var err error
			val, err = strconv.ParseFloat(valueStr, 64)
			if err != nil {
				return 0, fmt.Errorf("无效数值 %s", valueStr)
			}
		}
		// 注册事件处理器
		handlerKey := eventName + "|" + operator + "|" + fmt.Sprintf("%.2f", val)
		if vm.Program.EventHandlers == nil {
			vm.Program.EventHandlers = make(map[string]string)
		}
		vm.Program.EventHandlers[handlerKey] = label
		vm.Log(fmt.Sprintf("注册事件处理器: %s %s %f -> %s", eventName, operator, val, label))
		return 0, nil
	case InstrLog:
		if len(args) < 1 {
			return 0, errors.New("LOG 需要消息")
		}
		message := strings.Join(args, " ")
		vm.Log(message)
		return 0, nil
	case InstrJettison:
		vm.Log("执行分离操作")
		if vm.Satellite != nil {
			for i := range vm.Satellite.CargoBays {
				if len(vm.Satellite.CargoBays[i].Loaded) > 0 {
					vm.Log(fmt.Sprintf("货舱 %d 中有 %d 件货物待分离", 
						vm.Satellite.CargoBays[i].Index, 
						len(vm.Satellite.CargoBays[i].Loaded)))
					break
				}
			}
		}
		return 0, nil
	case InstrOrient:
		if len(args) < 1 {
			return 0, errors.New("ORIENT 需要目标")
		}
		target := args[0]
		vm.Log(fmt.Sprintf("姿态指向 %s", target))
		return 0, nil
	case InstrShutdown:
		vm.Log("系统关机")
		vm.IsRunning = false
		return 0, nil
	case InstrIf:
		if len(args) < 5 {
			return 0, errors.New("IF 需要5个参数: $REG 比较符 值 GOTO :label")
		}
		regName := strings.TrimPrefix(args[0], "$")
		operator := args[1]
		valueStr := args[2]
		if args[3] != "GOTO" && args[3] != "goto" {
			return 0, errors.New("IF 格式应为 IF $REG 比较符 值 GOTO :label")
		}
		label := strings.TrimPrefix(args[4], ":")
		regVal, ok := vm.Registers[regName]
		if !ok {
			return 0, fmt.Errorf("未找到寄存器 %s", regName)
		}
		var cmpVal float64
		if strings.HasPrefix(valueStr, "$") {
			reg := strings.TrimPrefix(valueStr, "$")
			if v, ok := vm.Registers[reg]; ok {
				cmpVal = v
			} else {
				return 0, fmt.Errorf("未找到寄存器 %s", reg)
			}
		} else {
			var err error
			cmpVal, err = strconv.ParseFloat(valueStr, 64)
			if err != nil {
				return 0, fmt.Errorf("无效数值 %s", valueStr)
			}
		}
		condition := false
		switch operator {
		case ">":
			condition = regVal > cmpVal
		case "<":
			condition = regVal < cmpVal
		case "=", "==":
			condition = regVal == cmpVal
		case ">=":
			condition = regVal >= cmpVal
		case "<=":
			condition = regVal <= cmpVal
		default:
			return 0, fmt.Errorf("未知比较符 %s", operator)
		}
		if condition {
			if dest, ok := vm.Labels[label]; ok {
				vm.InstructionPtr = dest
			} else {
				return 0, fmt.Errorf("未找到标签 %s", label)
			}
		}
		return 0, nil
	case InstrGoto:
		if len(args) < 1 {
			return 0, errors.New("GOTO 需要标签")
		}
		label := strings.TrimPrefix(args[0], ":")
		if dest, ok := vm.Labels[label]; ok {
			vm.InstructionPtr = dest
		} else {
			return 0, fmt.Errorf("未找到标签 %s", label)
		}
		return 0, nil
	case InstrCall:
		if len(args) < 1 {
			return 0, errors.New("CALL 需要标签")
		}
		label := strings.TrimPrefix(args[0], ":")
		if dest, ok := vm.Labels[label]; ok {
			vm.CallStack = append(vm.CallStack, vm.InstructionPtr)
			vm.InstructionPtr = dest
		} else {
			return 0, fmt.Errorf("未找到标签 %s", label)
		}
		return 0, nil
	case InstrRet:
		if len(vm.CallStack) == 0 {
			return 0, errors.New("RET 在没有调用的情况下执行")
		}
		returnAddr := vm.CallStack[len(vm.CallStack)-1]
		vm.CallStack = vm.CallStack[:len(vm.CallStack)-1]
		vm.InstructionPtr = returnAddr
		return 0, nil
	case InstrCalc:
		if len(args) < 3 {
			return 0, errors.New("CALC 格式: CALC <结果寄存器> <操作数1> <操作符> <操作数2>")
		}
		destReg := args[0]
		op1 := args[1]
		operator := args[2]
		if len(args) < 4 {
			return 0, errors.New("CALC 需要两个操作数")
		}
		op2 := args[3]
		var v1, v2 float64
		var err error
		if strings.HasPrefix(op1, "$") {
			reg := strings.TrimPrefix(op1, "$")
			if val, ok := vm.Registers[reg]; ok {
				v1 = val
			} else {
				return 0, fmt.Errorf("未找到寄存器 %s", reg)
			}
		} else {
			v1, err = strconv.ParseFloat(op1, 64)
			if err != nil {
				return 0, fmt.Errorf("无效数值 %s", op1)
			}
		}
		if strings.HasPrefix(op2, "$") {
			reg := strings.TrimPrefix(op2, "$")
			if val, ok := vm.Registers[reg]; ok {
				v2 = val
			} else {
				return 0, fmt.Errorf("未找到寄存器 %s", reg)
			}
		} else {
			v2, err = strconv.ParseFloat(op2, 64)
			if err != nil {
				return 0, fmt.Errorf("无效数值 %s", op2)
			}
		}
		var result float64
		switch operator {
		case "+":
			result = v1 + v2
		case "-":
			result = v1 - v2
		case "*":
			result = v1 * v2
		case "/":
			if v2 == 0 {
				return 0, errors.New("除以零")
			}
			result = v1 / v2
		default:
			return 0, fmt.Errorf("未知操作符 %s", operator)
		}
		vm.Registers[destReg] = result
		vm.Program.Registers[destReg] = result
		return 0, nil
	case InstrPing:
		target := "地面站"
		if len(args) > 0 {
			target = args[0]
		}
		// 模拟 ping 延迟（1-3秒），使用整数取模避免浮点取模错误
		delay := 1.0 + float64(len(target)%3)
		vm.Log(fmt.Sprintf("PING %s 延迟: %.2fs", target, delay))
		return delay, nil
	case InstrFire:
		if len(args) < 1 {
			return 0, errors.New("FIRE 需要推力百分比参数 (0-100)")
		}
		thrustPercent, err := strconv.ParseFloat(args[0], 64)
		if err != nil || thrustPercent < 0 || thrustPercent > 100 {
			return 0, errors.New("推力百分比必须为 0-100 的数字")
		}
		if vm.Satellite == nil {
			vm.Log("没有关联的航天器，无法点火")
			return 0, nil
		}
		dvCost := thrustPercent * 0.1
		if vm.Satellite.DeltaVRemaining < dvCost {
			vm.Log(fmt.Sprintf("Δv 不足! 需要 %.1f, 当前 %.1f", dvCost, vm.Satellite.DeltaVRemaining))
			return 0, nil
		}
		vm.Satellite.DeltaVRemaining -= dvCost
		vm.Log(fmt.Sprintf("引擎点火: 推力 %.0f%%, 消耗 Δv %.1f m/s, 剩余 %.1f m/s", 
			thrustPercent, dvCost, vm.Satellite.DeltaVRemaining))
		return 0, nil
	case InstrMeasure, "SAT_MEASURE":
		vm.Log("执行数据采集...")
		hasSensor := false
		for _, m := range vm.Satellite.Modules {
			if m.Type == "Sensor" {
				hasSensor = true
				break
			}
		}
		if !hasSensor {
			return 0, errors.New("没有传感器模块，无法采集数据")
		}
		dataAmount := 0.5
		vm.Satellite.DataStored += dataAmount
		vm.Log(fmt.Sprintf("采集 %.2f MB 数据，当前数据: %.2f MB", dataAmount, vm.Satellite.DataStored))
		return 0, nil
	case InstrSend, "SAT_SEND":
		if vm.Satellite.DataStored == 0 {
			vm.Log("没有数据可发送")
			return 0, nil
		}
		hasComms := false
		for _, m := range vm.Satellite.Modules {
			if m.Type == "Comms" {
				hasComms = true
				break
			}
		}
		if !hasComms {
			return 0, errors.New("没有通信模块，无法发送数据")
		}
		reward := int64(vm.Satellite.DataStored * 10)
		vm.Log(fmt.Sprintf("发送 %.2f MB 数据，获得 %d 信用点（模拟）", vm.Satellite.DataStored, reward))
		vm.Satellite.DataStored = 0
		return 0, nil
	default:
		return 0, fmt.Errorf("未知指令 %s", instr)
	}
}

// Log 记录一条日志
func (vm *ECCLVM) Log(msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s", timestamp, msg)
	vm.LastLog = append(vm.LastLog, entry)
	fmt.Println(entry)
}

// Step 执行一步
func (vm *ECCLVM) Step() error {
	if !vm.IsRunning {
		return errors.New("虚拟机未运行")
	}
	for vm.InstructionPtr < len(vm.Program.CodeLines) {
		line := vm.Program.CodeLines[vm.InstructionPtr]
		instr, args, err := parseInstruction(line)
		if err != nil {
			return err
		}
		if instr == "" {
			vm.InstructionPtr++
			continue
		}
		vm.InstructionPtr++
		wait, err := vm.executeInstruction(instr, args)
		if err != nil {
			return err
		}
		if wait > 0 {
			return nil
		}
	}
	vm.IsRunning = false
	return nil
}

// Run 运行程序
func (vm *ECCLVM) Run() error {
	vm.IsRunning = true
	for vm.IsRunning {
		err := vm.Step()
		if err != nil {
			return err
		}
	}
	return nil
}

// TriggerEvent 触发一个事件，如果有匹配的处理器则执行跳转
func (vm *ECCLVM) TriggerEvent(eventType string, value float64) error {
	if vm.Program.EventHandlers == nil {
		return nil
	}
	for key, label := range vm.Program.EventHandlers {
		parts := strings.Split(key, "|")
		if len(parts) != 3 {
			continue
		}
		evtType := parts[0]
		operator := parts[1]
		thresholdStr := parts[2]
		if evtType != eventType {
			continue
		}
		threshold, err := strconv.ParseFloat(thresholdStr, 64)
		if err != nil {
			continue
		}
		condition := false
		switch operator {
		case ">":
			condition = value > threshold
		case "<":
			condition = value < threshold
		case "=", "==":
			condition = value == threshold
		case ">=":
			condition = value >= threshold
		case "<=":
			condition = value <= threshold
		default:
			continue
		}
		if condition {
			if dest, ok := vm.Labels[label]; ok {
				vm.InstructionPtr = dest
				vm.Log(fmt.Sprintf("事件触发: %s = %.2f -> 跳转到 %s", eventType, value, label))
				return nil
			}
		}
	}
	return nil
}

// LoadProgram 加载程序并解析标签
func LoadProgram(prog *domain.ECCLProgram) error {
	lines := strings.Split(prog.Code, "\n")
	var codeLines []string
	labels := make(map[string]int)
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		codeLines = append(codeLines, line)
		if strings.HasPrefix(line, ":") {
			labelName := strings.TrimPrefix(line, ":")
			labels[labelName] = len(codeLines) - 1
		}
	}
	prog.CodeLines = codeLines
	prog.Labels = labels
	return nil
}
