// Copyright (c) Contributors to the Apptainer project, established as
//   Apptainer a Series of LF Projects LLC.
//   For website terms of use, trademark policy, privacy policy and other
//   project policies see https://lfprojects.org/policies
// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package apptainer

import (
	"fmt"
	"syscall"

	"github.com/apptainer/apptainer/pkg/sylog"
)

// OciUpdate updates container cgroups resources
func OciUpdate(containerID string, args *OciArgs) error {
	runcArgs := []string{
		"--root=" + OciStateDir,
		"update",
		"-r", args.FromFile,
		containerID,
	}

	sylog.Debugf("Calling runc with args %v", runcArgs)
	if err := syscall.Exec(runc, runcArgs, []string{}); err != nil {
		return fmt.Errorf("while calling runc: %w", err)
	}

	return nil
}
