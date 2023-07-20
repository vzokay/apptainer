// Copyright (c) Contributors to the Apptainer project, established as
//   Apptainer a Series of LF Projects LLC.
//   For website terms of use, trademark policy, privacy policy and other
//   project policies see https://lfprojects.org/policies
// Copyright (c) 2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"context"
	"fmt"

	"github.com/apptainer/apptainer/pkg/image"
	ocitypes "github.com/containers/image/v5/types"
)

// Push pushes an image into an OCI registry, as an OCI image (not an ORAS artifact).
// At present, only OCI-SIF images can be pushed in this manner.
func Push(ctx context.Context, sourceFile string, destRef string, ociAuth *ocitypes.DockerAuthConfig) error {
	img, err := image.Init(sourceFile, false)
	if err != nil {
		return err
	}
	defer img.File.Close()

	switch img.Type {
	case image.OCISIF:
		return pushOCISIF(ctx, sourceFile, destRef, ociAuth)
	case image.SIF:
		return fmt.Errorf("non OCI SIF images can only be pushed to OCI registries via oras://")
	}

	return fmt.Errorf("push only supports SIF images")
}
