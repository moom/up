// Ultimate Provisioner: UP cmd
// Copyright (c) 2019 Stephen Cheng and contributors

/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package impl

import (
	"github.com/fatih/color"
	ms "github.com/mitchellh/mapstructure"
	"github.com/stephencheng/up/model/core"
	u "github.com/stephencheng/up/utils"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path"
)

type CmdFuncAction struct {
	Do   interface{}
	Vars *core.Cache
	Cmds *CmdCmds
}

type CmdCmd struct {
	Name string
	Desc string
	Cmd  interface{}
}

type GeneralCmd struct {
	Name  string
	Value string
}

type CmdCmds []CmdCmd

func (f *CmdFuncAction) Adapt() {
	var cmds CmdCmds

	switch f.Do.(type) {

	case []interface{}:
		err := ms.Decode(f.Do, &cmds)
		u.LogErrorAndExit("Cmd adapter", err, "please fix cmd command configuration")

	default:
		u.LogWarn("cmd", "Not implemented or void for no action!")
	}

	f.Cmds = &cmds

}

func (cmdCmd *CmdCmd) runCmd(whichtype string, f func()) {
	invalidTypeHint := func(typeGot string) {
		u.LogWarn("type mismatch", u.Spf("cmd name: %s -> type wanted: %s, got :%s", cmdCmd.Name, whichtype, typeGot))
	}
	switch cmdCmd.Cmd.(type) {
	case string:
		if whichtype == "string" {
			f()
		} else {
			invalidTypeHint("string")
		}

	case int:
		if whichtype == "int" {
			f()
		} else {
			invalidTypeHint("int")
		}

	case map[interface{}]interface{}:
		if whichtype == "map" {
			f()
		} else {
			invalidTypeHint("map")
		}

	default:
		u.LogWarn("cmd", "Not implemented or void for no action!")
	}

}

func (f *CmdFuncAction) Exec() {

	//u.P("executing cmd commands")
	for idx, cmdItem := range *f.Cmds {
		//u.Pfv("cmd cmdItem(%2d): %s (%s)\n%s\n", idx+1, cmdItem.Name, cmdItem.Desc, color.HiBlueString("%s", cmdItem.Cmd))
		u.Pfv("cmd cmdItem(%2d): %s (%s)\n", idx+1, cmdItem.Name, cmdItem.Desc)
		u.Pfvv("%s\n", color.MagentaString("%s", cmdItem.Cmd))

		u.LogDesc("substep", cmdItem.Name, cmdItem.Desc)
		switch cmdItem.Name {
		case "print":
			cmdItem.runCmd("string", func() {
				cmdRendered := core.Render(cmdItem.Cmd.(string), f.Vars)
				u.Pfv("%s\n", color.HiGreenString("%s", cmdRendered))
			})

		case "printobj":
			u.Dvvvv(cmdItem.Cmd)
			cmdItem.runCmd("string", func() {
				objname := core.Render(cmdItem.Cmd.(string), f.Vars)
				//obj := cache.RuntimeVarsAndDvarsMerged.Get(objname)
				obj := f.Vars.Get(objname)
				u.Ppfmsg(u.Spf("object:\n %s", objname), obj)
			})

		case "dereg":
			cmdItem.runCmd("string", func() {
				varname := core.Render(cmdItem.Cmd.(string), f.Vars)
				u.Pfv("deregister var: %s\n", color.HiGreenString("%s", varname))
				core.RuntimeVarsAndDvarsMerged.Delete(varname)
				f.Vars.Delete(varname)
			})
			u.Ppmsgvvvvvhint("after reg the var - global:", core.RuntimeVarsAndDvarsMerged)
			u.Ppmsgvvvvvhint("after reg the var - local:", f.Vars)

		case "sleep":
			cmdItem.runCmd("int", func() {
				mscnt := cmdItem.Cmd.(int)
				u.Sleep(mscnt)
			})

		case "readfile":
			cmdItem.runCmd("map", func() {
				cmd := cmdItem.Cmd.(map[interface{}]interface{})
				var varname, filename, dir, raw string
				var localonly bool
				for k, v := range cmd {
					switch k.(string) {
					case "reg":
						raw = v.(string)
						varname = core.Render(raw, f.Vars)
					case "filename":
						raw = v.(string)
						filename = core.Render(raw, f.Vars)
					case "dir":
						raw = v.(string)
						dir = core.Render(raw, f.Vars)
					case "localonly":
						localonly = v.(bool)
					}
				}
				filepath := path.Join(dir, filename)

				content, err := ioutil.ReadFile(filepath)
				u.LogErrorAndExit("cmd readfile", err, "please fix file path and name issues")

				if localonly {
					f.Vars.Put(varname, string(content))
				} else {
					core.RuntimeVarsAndDvarsMerged.Put(varname, string(content))
					f.Vars.Put(varname, string(content))
				}

			})

			u.Ppmsgvvvvvhint("after reg the var - global:", core.RuntimeVarsAndDvarsMerged)
			u.Ppmsgvvvvvhint("after reg the var - local:", f.Vars)

		case "writefile":
			cmdItem.runCmd("map", func() {
				cmd := cmdItem.Cmd.(map[interface{}]interface{})
				var content, filename, dir, raw string
				for k, v := range cmd {
					switch k.(string) {
					case "content":
						contentRaw := v.(string)
						content = core.Render(contentRaw, f.Vars)
					case "filename":
						raw = v.(string)
						filename = core.Render(raw, f.Vars)
					case "dir":
						raw = v.(string)
						dir = core.Render(raw, f.Vars)
					}
				}
				filepath := path.Join(dir, filename)
				ioutil.WriteFile(filepath, []byte(content), 0644)
			})

		case "template":
			cmdItem.runCmd("map", func() {
				cmd := cmdItem.Cmd.(map[interface{}]interface{})
				var src, dest, raw, datakey, rendered string
				var data interface{}
				for k, v := range cmd {
					switch k.(string) {
					case "src":
						raw = v.(string)
						src = core.Render(raw, f.Vars)
					case "data":
						raw = v.(string)
						datakey = core.Render(raw, f.Vars)
						data = f.Vars.Get(datakey)
					case "dest":
						raw = v.(string)
						dest = core.Render(raw, f.Vars)
					}
				}

				tbuf, err := ioutil.ReadFile(src)
				if data == nil || data == "" {
					rendered = core.Render(string(tbuf), f.Vars)
				} else {
					rendered = core.Render(string(tbuf), data)
				}

				u.LogErrorAndExit("cmd template", err, "please fix file path and name issues")
				ioutil.WriteFile(dest, []byte(rendered), 0644)
			})

		case "reg":
			cmdItem.runCmd("map", func() {
				regCmd := cmdItem.Cmd.(map[interface{}]interface{})
				var varname, varvalue string
				var localonly bool
				for k, v := range regCmd {
					if k.(string) == "name" {
						varname = v.(string)
					}
					if k.(string) == "value" {
						varvalueRaw := v.(string)
						varvalue = core.Render(varvalueRaw, f.Vars)
					}
					if k.(string) == "localonly" {
						localonly = v.(bool)
					}
				}

				if localonly {
					f.Vars.Put(varname, varvalue)
				} else {
					core.RuntimeVarsAndDvarsMerged.Put(varname, varvalue)
					f.Vars.Put(varname, varvalue)
				}

			})
			u.Ppmsgvvvvvhint("after reg the var - global:", core.RuntimeVarsAndDvarsMerged)
			u.Ppmsgvvvvvhint("after reg the var - local:", f.Vars)
		case "to_object":
			//src: a var name to get the yml content from
			//reg: a registered name to cache the variable
			//localonly: if set, then the variable will not be saved to global space
			cmdItem.runCmd("map", func() {
				cmd := cmdItem.Cmd.(map[interface{}]interface{})
				var key, src, reg string
				var localonly bool
				for k, v := range cmd {
					if k.(string) == "key" {
						keyRaw := v.(string)
						key = core.Render(keyRaw, f.Vars)
					}
					if k.(string) == "src" {
						srcRaw := v.(string)
						src = core.Render(srcRaw, f.Vars)
					}
					if k.(string) == "reg" {
						regRaw := v.(string)
						reg = core.Render(regRaw, f.Vars)
					}
					if k.(string) == "localonly" {
						localonly = v.(bool)
					}
				}

				srcyml := func() string {
					if src != "" && key != "" {
						u.InvalidAndExit("locate yml string", "you can only use either key or src, but not both")
					}
					if src != "" {
						return src
					}
					if key != "" {
						t := f.Vars.Get(key)
						if t != nil {
							return t.(string)
						} else {
							u.InvalidAndExit("locate yml string", "please use a valid addressable key to locate a yml document")
							return ""
						}
					}
					return ""
				}()
				obj := new(interface{})
				err := yaml.Unmarshal([]byte(srcyml), obj)
				u.LogErrorAndExit("cmd to_object:", err, "please validate the ymal content")

				if localonly {
					(*f.Vars).Put(reg, *obj)
				} else {
					core.RuntimeVarsAndDvarsMerged.Put(src, reg)
					(*f.Vars).Put(reg, *obj)
				}

			})
			u.Ppmsgvvvvvhint("after reg the var - global:", core.RuntimeVarsAndDvarsMerged)
			u.Ppmsgvvvvvhint("after reg the var - local:", f.Vars)
		default:
			u.Pferror("warrning: check cmd name:(%s),%s\n", cmdItem.Name, "cmd not implemented")
		}

	}
}

