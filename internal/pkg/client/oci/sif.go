// Copyright (c) Contributors to the Apptainer project, established as
//   Apptainer a Series of LF Projects LLC.
//   For website terms of use, trademark policy, privacy policy and other
//   project policies see https://lfprojects.org/policies
// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2018-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"context"
	"fmt"
	"reflect"

	"github.com/apptainer/apptainer/internal/pkg/build"
	"github.com/apptainer/apptainer/internal/pkg/build/oci"
	"github.com/apptainer/apptainer/internal/pkg/cache"
	buildtypes "github.com/apptainer/apptainer/pkg/build/types"
	"github.com/apptainer/apptainer/pkg/sylog"
)

// pullSif will build a SIF image into the cache if directTo="", or a specific file if directTo is set.
func pullSif(ctx context.Context, imgCache *cache.Handle, directTo, pullFrom string, opts PullOptions) (imagePath string, err error) {
	sys := sysCtx(opts)
	if opts.Pullarch != "" {
		if arch, ok := oci.ArchMap[opts.Pullarch]; ok {
			sys.ArchitectureChoice = arch.Arch
			sys.VariantChoice = arch.Var
		} else {
			keys := reflect.ValueOf(oci.ArchMap).MapKeys()
			return "", fmt.Errorf("failed to parse the arch value: %s, should be one of %v", opts.Pullarch, keys)
		}
	}
	hash, err := oci.ImageDigest(ctx, pullFrom, sys)
	if err != nil {
		return "", fmt.Errorf("failed to get checksum for %s: %s", pullFrom, err)
	}

	if directTo != "" {
		sylog.Infof("Converting OCI blobs to SIF format")
		if err := convertOciToSIF(ctx, imgCache, pullFrom, directTo, opts); err != nil {
			return "", fmt.Errorf("while building SIF from layers: %v", err)
		}
		imagePath = directTo
	} else {

		cacheEntry, err := imgCache.GetEntry(cache.OciTempCacheType, hash)
		if err != nil {
			return "", fmt.Errorf("unable to check if %v exists in cache: %v", hash, err)
		}
		defer cacheEntry.CleanTmp()
		if !cacheEntry.Exists {
			sylog.Infof("Converting OCI blobs to SIF format")

			if err := convertOciToSIF(ctx, imgCache, pullFrom, cacheEntry.TmpPath, opts); err != nil {
				return "", fmt.Errorf("while building SIF from layers: %v", err)
			}

			err = cacheEntry.Finalize()
			if err != nil {
				return "", err
			}

		} else {
			sylog.Infof("Using cached SIF image")
		}
		imagePath = cacheEntry.Path
	}

	return imagePath, nil
}

// convertOciToSIF will convert an OCI source into a SIF using the build routines
func convertOciToSIF(ctx context.Context, imgCache *cache.Handle, image, cachedImgPath string, opts PullOptions) error {
	if imgCache == nil {
		return fmt.Errorf("image cache is undefined")
	}

	b, err := build.NewBuild(
		image,
		build.Config{
			Dest:      cachedImgPath,
			Format:    "sif",
			NoCleanUp: opts.NoCleanUp,
			Opts: buildtypes.Options{
				TmpDir:           opts.TmpDir,
				NoCache:          imgCache.IsDisabled(),
				NoTest:           true,
				NoHTTPS:          opts.NoHTTPS,
				DockerAuthConfig: opts.OciAuth,
				DockerDaemonHost: opts.DockerHost,
				ImgCache:         imgCache,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("unable to create new build: %v", err)
	}

	return b.Full(ctx)
}
