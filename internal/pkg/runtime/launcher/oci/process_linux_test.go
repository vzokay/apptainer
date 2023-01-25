// Copyright (c) Contributors to the Apptainer project, established as
//   Apptainer a Series of LF Projects LLC.
//   For website terms of use, trademark policy, privacy policy and other
//   project policies see https://lfprojects.org/policies
// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"os"
	"reflect"
	"testing"
)

func TestApptainerEnvMap(t *testing.T) {
	tests := []struct {
		name   string
		setEnv map[string]string
		want   map[string]string
	}{
		{
			name:   "None",
			setEnv: map[string]string{},
			want:   map[string]string{},
		},
		{
			name:   "NonPrefixed",
			setEnv: map[string]string{"FOO": "bar"},
			want:   map[string]string{},
		},
		{
			name:   "PrefixedSingle",
			setEnv: map[string]string{"APPTAINERENV_FOO": "bar"},
			want:   map[string]string{"FOO": "bar"},
		},
		{
			name: "PrefixedMultiple",
			setEnv: map[string]string{
				"APPTAINERENV_FOO": "bar",
				"APPTAINERENV_ABC": "123",
			},
			want: map[string]string{
				"FOO": "bar",
				"ABC": "123",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.setEnv {
				os.Setenv(k, v)
				t.Cleanup(func() {
					os.Unsetenv(k)
				})
			}
			if got := apptainerEnvMap(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("apptainerEnvMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
