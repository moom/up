// Ultimate Provisioner: UP cmd
// Copyright (c) 2019 Stephen Cheng and contributors

/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package impl

import (
	"github.com/imdario/mergo"
	ms "github.com/mitchellh/mapstructure"
	"github.com/mohae/deepcopy"
	"github.com/stephencheng/up/biz"
	"github.com/stephencheng/up/model/core"
	u "github.com/stephencheng/up/utils"
	ee "github.com/stephencheng/up/utils/error"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type Step struct {
	Name   string
	Do     interface{} //FuncImpl
	Func   string
	Vars   core.Cache
	Dvars  core.Dvars
	Desc   string
	Reg    string
	Flags  []string
	If     string
	Else   interface{}
	Loop   interface{}
	Until  string
	RefDir string
}

type Steps []Step

/*
ExecbaseVars is the the scope containing passed in caller vars
local vars will be merged depending if it is a callee task
merge taskvars before final merge as task vars will be the calculated result of current stack
*/
func (step *Step) getRuntimeExecVars(fromBlock bool) *core.Cache {
	var execvars core.Cache
	var resultVars *core.Cache

	execvars = deepcopy.Copy(*core.TaskRuntime().ExecbaseVars).(core.Cache)
	//u.Ptmpdebug("22", "others get runtime")
	//u.Ptmpdebug("11", execvars)
	taskVars := core.TaskRuntime().TaskVars
	mergo.Merge(&execvars, taskVars, mergo.WithOverride)
	//u.Ptmpdebug("33", execvars)
	//u.Ptmpdebug("44", step.Vars)

	if fromBlock {
		blockvars := core.BlockStack.GetTop().(*core.BlockRuntimeContext).BlockBaseVars
		mergo.Merge(&execvars, blockvars, mergo.WithOverride)
		mergo.Merge(&execvars, &step.Vars, mergo.WithOverride)
		resultVars = &execvars
	} else {
		if IsCalledTask() {
			//u.Ptmpdebug("if", "if")
			if step.Vars != nil {
				mergo.Merge(&step.Vars, &execvars, mergo.WithOverride)
				resultVars = &step.Vars
			} else {
				resultVars = &execvars
			}
		} else {
			//u.Ptmpdebug("else", "else")
			mergo.Merge(&execvars, &step.Vars, mergo.WithOverride)
			resultVars = &execvars
		}
	}
	u.Pfvvvv("current exec runtime vars:")
	u.Ppmsgvvvv(resultVars)
	//u.Ptmpdebug("55", resultVars)

	//so far the execvars includes: scope vars + scope dvars + global runtime vars + task vars
	resultVars = core.VarsMergedWithDvars("local", resultVars, &step.Dvars, resultVars)

	//so far the resultVars includes: the local vars + dvars rendered using execvars
	u.Ppmsgvvvhint("overall final exec vars:", resultVars)
	//u.Ptmpdebug("99", resultVars)
	return resultVars
}

type LoopItem struct {
	Index  int
	Index1 int
	Item   interface{}
}

func chainAction(action *biz.Do) {
	(*action).Adapt()
	(*action).Exec()
}

func validation(vars *core.Cache) {
	identified := false

	for k, _ := range *vars {
		if k == "" {
			u.InvalidAndExit("validating var name", "var name can not be empty")
		}
		if u.CharIsNum(k[0:1]) != -1 {
			identified = true
			u.InvalidAndExit("validating var name", u.Spf("var name (%s) can not start with number", k))
		}
	}

	if identified {
		u.LogError("vars validation", "please fix all validation before continue")
		os.Exit(-1)
	}

}

func (step *Step) Exec(fromBlock bool) {
	var action biz.Do

	var bizErr *ee.Error = ee.New()
	var stepExecVars *core.Cache
	stepExecVars = step.getRuntimeExecVars(fromBlock)
	//u.Ptmpdebug("99", stepExecVars)
	validation(stepExecVars)

	if step.Flags != nil && u.Contains(step.Flags, "pause") {
		pause(stepExecVars)
	}

	routeFuncType := func(loopItem *LoopItem) {
		if loopItem != nil {
			stepExecVars.Put("loopitem", loopItem.Item)
			stepExecVars.Put("loopindex", loopItem.Index)
			stepExecVars.Put("loopindex1", loopItem.Index1)
		}

		switch step.Func {
		case FUNC_SHELL:
			funcAction := ShellFuncAction{
				Do:   step.Do,
				Vars: stepExecVars,
			}
			action = biz.Do(&funcAction)

		case FUNC_CALL:
			funcAction := CallFuncAction{
				Do:   step.Do,
				Vars: stepExecVars,
			}
			action = biz.Do(&funcAction)

		case FUNC_BLOCK:
			funcAction := BlockFuncAction{
				Do:   step.Do,
				Vars: stepExecVars,
			}
			action = biz.Do(&funcAction)

		case FUNC_CMD:
			funcAction := CmdFuncAction{
				Do:   step.Do,
				Vars: stepExecVars,
			}
			action = biz.Do(&funcAction)

		case "":
			u.InvalidAndExit("Step dispatch", "func name is empty and not defined")
			bizErr.Mark = "func name not implemented"

		default:
			u.InvalidAndExit("Step dispatch", u.Spf("func name(%s) is not recognised and implemented", step.Func))
			bizErr.Mark = "func name not implemented"
		}
	}

	dryRunOrContinue := func() {
		//example to stop further steps
		//f := u.MustConditionToContinueFunc(func() bool {
		//	return action != nil
		//})
		//
		//u.DryRunOrExit("Step Exec", f, "func name must be valid")

		alloweErrors := []string{
			"func name not implemented",
		}

		DryRunAndSkip(
			bizErr.Mark,
			alloweErrors,
			ContinueFunc(
				func() {
					if step.Loop != nil {
						rawUtil := step.Until
						func() {
							//loop points to a var name which is a slice
							if reflect.TypeOf(step.Loop).Kind() == reflect.String {
								loopVarName := core.Render(step.Loop.(string), stepExecVars)
								loopObj := stepExecVars.Get(loopVarName)
								if loopObj == nil {
									u.InvalidAndExit("Evaluating loop var and object", u.Spf("Please use a correct varname:(%s) containing a list of values", loopVarName))
								}
								if reflect.TypeOf(loopObj).Kind() == reflect.Slice {
									switch loopObj.(type) {
									case []interface{}:
										for idx, item := range loopObj.([]interface{}) {
											routeFuncType(&LoopItem{idx, idx + 1, item})
											if rawUtil != "" {
												untilEval := core.Render(rawUtil, stepExecVars)
												toBreak, err := strconv.ParseBool(untilEval)
												u.LogErrorAndExit("evaluate until condition", err, u.Spf("please fix until condition evaluation: [%s]", untilEval))
												if toBreak {
													u.Pvvvv("loop util conditional break")
													break
												} else {
													chainAction(&action)
												}
											} else {
												chainAction(&action)
											}
										}

									case []string:
										for idx, item := range loopObj.([]string) {
											routeFuncType(&LoopItem{idx, idx + 1, item})
											if rawUtil != "" {
												untilEval := core.Render(rawUtil, stepExecVars)
												toBreak, err := strconv.ParseBool(untilEval)
												u.LogErrorAndExit("evaluate until condition", err, u.Spf("please fix until condition evaluation: [%s]", untilEval))
												if toBreak {
													u.Pvvvv("loop util conditional break")
													break
												} else {
													chainAction(&action)
												}
											} else {
												chainAction(&action)
											}
										}

									default:
										u.LogWarn("loop item evaluation", "Loop item type is not supported yet!")
									}
								} else {
									u.InvalidAndExit("evaluate loop var", "loop var is not a array/list/slice")
								}
							} else if reflect.TypeOf(step.Loop).Kind() == reflect.Slice {
								//loop itself is a slice
								for idx, item := range step.Loop.([]interface{}) {
									routeFuncType(&LoopItem{idx, idx + 1, item})
									if rawUtil != "" {
										untilEval := core.Render(rawUtil, stepExecVars)
										toBreak, err := strconv.ParseBool(untilEval)
										u.LogErrorAndExit("evaluate until condition", err, u.Spf("please fix until condition evaluation: [%s]", untilEval))
										if toBreak {
											u.Pvvvv("loop util conditional break")
											break
										} else {
											chainAction(&action)
										}
									} else {
										chainAction(&action)
									}
								}
							} else {
								u.InvalidAndExit("evaluate loop items", "please either use a list or a template evaluation which could result in a value of a list")
							}
						}()

					} else {
						routeFuncType(nil)
						chainAction(&action)
					}

				}),
			nil,
		)
	}

	func() {
		if step.If != "" {
			IfEval := core.Render(step.If, stepExecVars)
			if IfEval != "<no value>" {
				goahead, err := strconv.ParseBool(IfEval)
				u.LogErrorAndExit("evaluate condition", err, u.Spf("please fix if condition evaluation: [%s]", IfEval))
				if goahead {
					dryRunOrContinue()
				} else {
					if step.Else != nil && step.Else != "" {
						doElse(step.Else, stepExecVars)
					} else {
						u.Pvvv("condition failed, skip executing step", step.Name)
					}
				}
			} else {
				u.Pvvv("condition failed, skip executing step", step.Name)
			}
		} else {
			dryRunOrContinue()
		}

	}()

}

func doElse(elseCalls interface{}, execVars *core.Cache) {
	var taskname string
	var tasknames []string
	var flow Steps

	u.Ptmpdebug("haha", "else .....")
	switch elseCalls.(type) {
	case string:
		taskname = elseCalls.(string)
		tasknames = append(tasknames, taskname)

	case []interface{}:
		elseStr := u.Spf("%s", elseCalls)
		u.Ptmpdebug("kak", elseStr)
		if strings.Index(elseStr, "map") != -1 && strings.Index(elseStr, "func:") != -1 {
			err := ms.Decode(elseCalls, &flow)
			u.LogErrorAndExit("load steps in else", err, "steps has configuration problem, please fix it")
			BlockFlowRun(&flow, execVars)
		} else {
			err := ms.Decode(elseCalls, &tasknames)
			u.LogErrorAndExit("call func alias: else", err, "please ref to a task name only")
		}

	default:
		u.LogWarn("else ..", "Not implemented or void for no action!")
	}

	if len(tasknames) > 0 {
		for _, tmptaskname := range tasknames {
			taskname := core.Render(tmptaskname, execVars)
			u.PpmsgvvvvvhintHigh(u.Spf("else caller vars to task (%s):", taskname), execVars)
			ExecTask(taskname, execVars)
		}
	}

}

func (steps *Steps) Exec(fromBlock bool) {

	for idx, step := range *steps {

		taskLayerCnt := core.TaskStack.GetLen()
		u.LogDesc("step", idx+1, taskLayerCnt, step.Name, step.Desc)
		u.Ppmsgvvvv(step)

		execStep := func() {
			rtContext := core.StepRuntimeContext{
				Stepname: step.Name,
			}
			core.StepStack.Push(&rtContext)

			//TODO: consider move task vars merging to here
			step.Exec(fromBlock)

			result := core.StepStack.GetTop().(*core.StepRuntimeContext).Result
			taskname := core.TaskStack.GetTop().(*core.TaskRuntimeContext).Taskname

			//TODO: add support for block
			if u.Contains([]string{FUNC_SHELL, FUNC_CALL}, step.Func) {
				if step.Reg == "auto" {
					core.TaskRuntime().ExecbaseVars.Put(u.Spf("register_%s_%s", taskname, step.Name), result.Output)
				} else if step.Reg != "" {
					core.TaskRuntime().ExecbaseVars.Put(u.Spf("%s", step.Reg), result.Output)
				}
				if step.Func == FUNC_SHELL {
					core.TaskRuntime().ExecbaseVars.Put("last_result", result)
				}

			}

			func() {
				result := core.StepStack.GetTop().(*core.StepRuntimeContext).Result

				if result != nil && result.Code == 0 {
					u.LogOk(".")
				}

				if !u.Contains(step.Flags, "ignore_error") {
					if result != nil && result.Code != 0 {
						u.InvalidAndExit("Failed And Not Ignored!", "You may want to continue and ignore the error")
					}
				} else {
					if result != nil && result.Code != 0 {
						u.LogWarn("HightLight:", "Error ignored!!!")
					}
				}

			}()

			core.StepStack.Pop()
		}

		if !core.TaskBreak {
			execStep()
		} else {
			core.TaskBreak = false
			break
		}

	}

}

