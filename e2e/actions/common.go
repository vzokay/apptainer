// Copyright (c) Contributors to the Apptainer project, established as
//   Apptainer a Series of LF Projects LLC.
//   For website terms of use, trademark policy, privacy policy and other
//   project policies see https://lfprojects.org/policies
// Copyright (c) 2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package actions

import (
	"os"
	"testing"

	"github.com/apptainer/apptainer/e2e/internal/e2e"
	"github.com/apptainer/apptainer/internal/pkg/util/fs"
)

func mkWorkspaceDirs(t *testing.T, hostCanaryDir, hostHomeDir, hostWorkDir, hostCanaryFile, hostCanaryFileWithComma, hostCanaryFileWithColon string) {
	e2e.Privileged(func(t *testing.T) {
		if err := os.RemoveAll(hostCanaryDir); err != nil && !os.IsNotExist(err) {
			t.Fatalf("failed to delete canary_dir: %s", err)
		}
		if err := os.RemoveAll(hostHomeDir); err != nil && !os.IsNotExist(err) {
			t.Fatalf("failed to delete workspace home: %s", err)
		}
		if err := os.RemoveAll(hostWorkDir); err != nil && !os.IsNotExist(err) {
			t.Fatalf("failed to delete workspace home: %s", err)
		}
	})(t)

	if err := fs.Mkdir(hostCanaryDir, 0o777); err != nil {
		t.Fatalf("failed to create canary_dir: %s", err)
	}
	if err := fs.Touch(hostCanaryFile); err != nil {
		t.Fatalf("failed to create canary_file: %s", err)
	}
	if err := fs.Touch(hostCanaryFileWithComma); err != nil {
		t.Fatalf("failed to create canary_file_comma: %s", err)
	}
	if err := fs.Touch(hostCanaryFileWithColon); err != nil {
		t.Fatalf("failed to create canary_file_colon: %s", err)
	}
	if err := os.Chmod(hostCanaryFile, 0o777); err != nil {
		t.Fatalf("failed to apply permissions on canary_file: %s", err)
	}
	if err := fs.Mkdir(hostHomeDir, 0o777); err != nil {
		t.Fatalf("failed to create workspace home directory: %s", err)
	}
	if err := fs.Mkdir(hostWorkDir, 0o777); err != nil {
		t.Fatalf("failed to create workspace home directory: %s", err)
	}
}
