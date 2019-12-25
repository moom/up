// Ultimate Provisioner: UP cmd
// Copyright (c) 2019 Stephen Cheng and contributors

/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package model

type Tasks []Task

type Task struct {
	Task interface{} //Steps
	Desc string
	Name string
}

type Steps []Step

type Step struct {
	Do   interface{} //FuncImpl
	Func string
	Desc string
}

type ShellCmds []string

