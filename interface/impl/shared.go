// Ultimate Provisioner: UP cmd
// Copyright (c) 2019 Stephen Cheng and contributors

/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package impl

const (
	FUNC_SHELL    = "shell"
	FUNC_TASK_REF = "task_ref"
)

type ExecResult struct {
	Code   int
	Output string
	ErrMsg string
}

