// Copyright 2019 Graham Clark. All rights reserved.  Use of this source
// code is governed by the MIT license that can be found in the LICENSE
// file.

// +build !windows
// +build arm arm64

package system

import "syscall"

func Dup2(oldfd int, newfd int) error {
	return syscall.Dup3(oldfd, newfd, 0)
}

//======================================================================
// Local Variables:
// mode: Go
// fill-column: 110
// End:
