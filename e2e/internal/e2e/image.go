// Copyright (c) Contributors to the Apptainer project, established as
//   Apptainer a Series of LF Projects LLC.
//   For website terms of use, trademark policy, privacy policy and other
//   project policies see https://lfprojects.org/policies
// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"context"
	"io"
	"net/http"
	"os"
	"runtime"
	"sync"
	"testing"

	useragent "github.com/apptainer/apptainer/pkg/util/user-agent"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
)

const ociArchiveURI = "https://github.com/apptainer/apptainer/releases/download/v0.1.0/alpine-oci-archive.tar"

var (
	ensureMutex sync.Mutex
	pullMutex   sync.Mutex
)

// EnsureImage checks if e2e test image is already built or built
// it otherwise.
func EnsureImage(t *testing.T, env TestEnv) {
	ensureMutex.Lock()
	defer ensureMutex.Unlock()

	switch _, err := os.Stat(env.ImagePath); {
	case err == nil:
		// OK: file exists, return
		return

	case os.IsNotExist(err):
		// OK: file does not exist, continue

	default:
		// FATAL: something else is wrong
		t.Fatalf("Failed when checking image %q: %+v\n",
			env.ImagePath,
			err)
	}

	env.RunApptainer(
		t,
		WithProfile(RootProfile),
		WithCommand("build"),
		WithArgs("--force", env.ImagePath, "testdata/Apptainer"),
		ExpectExit(0),
	)
}

// EnsureSingularityImage checks if e2e test singularity image is already
// built or built it otherwise.
func EnsureSingularityImage(t *testing.T, env TestEnv) {
	ensureMutex.Lock()
	defer ensureMutex.Unlock()

	switch _, err := os.Stat(env.SingularityImagePath); {
	case err == nil:
		// OK: file exists, return
		return

	case os.IsNotExist(err):
		// OK: file does not exist, continue

	default:
		// FATAL: something else is wrong
		t.Fatalf("Failed when checking image %q: %+v\n",
			env.SingularityImagePath,
			err)
	}

	env.RunApptainer(
		t,
		WithProfile(RootProfile),
		WithCommand("build"),
		WithArgs("--force", env.SingularityImagePath, "testdata/Singularity_legacy.def"),
		ExpectExit(0),
	)
}

var orasImageOnce sync.Once

func EnsureORASImage(t *testing.T, env TestEnv) {
	EnsureImage(t, env)

	ensureMutex.Lock()
	defer ensureMutex.Unlock()

	orasImageOnce.Do(func() {
		env.RunApptainer(
			t,
			WithProfile(UserProfile),
			WithCommand("push"),
			WithArgs(env.ImagePath, env.OrasTestImage),
			ExpectExit(0),
		)
		if t.Failed() {
			t.Fatalf("failed to push ORAS image to local registry")
		}
	})
}

// PullImage will pull a test image.
func PullImage(t *testing.T, env TestEnv, imageURL string, arch string, path string) {
	pullMutex.Lock()
	defer pullMutex.Unlock()

	if arch == "" {
		arch = runtime.GOARCH
	}

	switch _, err := os.Stat(path); {
	case err == nil:
		// OK: file exists, return
		return

	case os.IsNotExist(err):
		// OK: file does not exist, continue

	default:
		// FATAL: something else is wrong
		t.Fatalf("Failed when checking image %q: %+v\n", path, err)
	}

	env.RunApptainer(
		t,
		WithProfile(UserProfile),
		WithCommand("pull"),
		WithArgs("--force", "--allow-unsigned", "--arch", arch, path, imageURL),
		ExpectExit(0),
	)
}

func CopyImage(t *testing.T, source, dest string, insecureSource, insecureDest bool) {
	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	policyCtx, err := signature.NewPolicyContext(policy)
	if err != nil {
		t.Fatalf("failed to copy %s to %s: %s", source, dest, err)
	}

	srcCtx := &types.SystemContext{
		OCIInsecureSkipTLSVerify:    insecureSource,
		DockerInsecureSkipTLSVerify: types.NewOptionalBool(insecureSource),
		DockerRegistryUserAgent:     useragent.Value(),
	}
	dstCtx := &types.SystemContext{
		OCIInsecureSkipTLSVerify:    insecureDest,
		DockerInsecureSkipTLSVerify: types.NewOptionalBool(insecureDest),
		DockerRegistryUserAgent:     useragent.Value(),
	}

	srcRef, err := docker.ParseReference("//" + source)
	if err != nil {
		t.Fatalf("failed to parse %s reference: %s", source, err)
	}
	dstRef, err := docker.ParseReference("//" + dest)
	if err != nil {
		t.Fatalf("failed to parse %s reference: %s", dest, err)
	}

	_, err = copy.Image(context.Background(), policyCtx, dstRef, srcRef, &copy.Options{
		ReportWriter:   io.Discard,
		SourceCtx:      srcCtx,
		DestinationCtx: dstCtx,
	})
	if err != nil {
		t.Fatalf("failed to copy %s to %s: %s", source, dest, err)
	}
}

func DownloadFile(url string, path string) error {
	dl, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dl.Close()

	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	_, err = io.Copy(dl, r.Body)
	if err != nil {
		return err
	}
	return nil
}

// EnsureImage checks if e2e OCI test image is available, and fetches
// it otherwise.
func EnsureOCIImage(t *testing.T, env TestEnv) {
	ensureMutex.Lock()
	defer ensureMutex.Unlock()

	switch _, err := os.Stat(env.OCIImagePath); {
	case err == nil:
		// OK: file exists, return
		return

	case os.IsNotExist(err):
		// OK: file does not exist, continue

	default:
		// FATAL: something else is wrong
		t.Fatalf("Failed when checking image %q: %+v\n",
			env.OCIImagePath,
			err)
	}

	// Prepare oci-archive source
	err := DownloadFile(ociArchiveURI, env.OCIImagePath)
	if err != nil {
		t.Fatalf("Could not download oci archive test file: %v", err)
	}
}
