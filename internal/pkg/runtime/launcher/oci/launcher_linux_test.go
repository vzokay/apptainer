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
	"reflect"
	"testing"

	"github.com/apptainer/apptainer/internal/pkg/runtime/launcher"
	"github.com/apptainer/apptainer/pkg/util/apptainerconf"
)

func TestNewLauncher(t *testing.T) {
	sc, err := apptainerconf.GetConfig(nil)
	if err != nil {
		t.Fatalf("while initializing apptainerconf: %s", err)
	}
	apptainerconf.SetCurrentConfig(sc)

	tests := []struct {
		name    string
		opts    []launcher.Option
		want    *Launcher
		wantErr bool
	}{
		{
			name:    "default",
			want:    &Launcher{apptainerConf: sc},
			wantErr: false,
		},
		{
			name: "validOption",
			opts: []launcher.Option{
				launcher.OptHome("/home/test", false, false),
			},
			want: &Launcher{cfg: launcher.Options{HomeDir: "/home/test"}, apptainerConf: sc},
		},
		{
			name: "unsupportedOption",
			opts: []launcher.Option{
				launcher.OptCacheDisabled(true),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLauncher(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLauncher() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewLauncher() = %v, want %v", got, tt.want)
			}
		})
	}
}
