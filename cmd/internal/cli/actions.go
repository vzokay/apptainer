// Copyright (c) Contributors to the Apptainer project, established as
//   Apptainer a Series of LF Projects LLC.
//   For website terms of use, trademark policy, privacy policy and other
//   project policies see https://lfprojects.org/policies
// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2018-2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/apptainer/apptainer/docs"
	"github.com/apptainer/apptainer/internal/pkg/cache"
	"github.com/apptainer/apptainer/internal/pkg/client/library"
	"github.com/apptainer/apptainer/internal/pkg/client/net"
	"github.com/apptainer/apptainer/internal/pkg/client/oci"
	"github.com/apptainer/apptainer/internal/pkg/client/oras"
	"github.com/apptainer/apptainer/internal/pkg/client/shub"
	"github.com/apptainer/apptainer/internal/pkg/runtime/launcher"
	"github.com/apptainer/apptainer/internal/pkg/runtime/launcher/native"
	ocilauncher "github.com/apptainer/apptainer/internal/pkg/runtime/launcher/oci"
	"github.com/apptainer/apptainer/internal/pkg/util/env"
	"github.com/apptainer/apptainer/internal/pkg/util/uri"
	"github.com/apptainer/apptainer/pkg/sylog"
	"github.com/spf13/cobra"
)

const (
	defaultPath = "/bin:/usr/bin:/sbin:/usr/sbin:/usr/local/bin:/usr/local/sbin"
)

func getCacheHandle(cfg cache.Config) *cache.Handle {
	envKey := env.TrimApptainerKey(cache.DirEnv)
	h, err := cache.New(cache.Config{
		ParentDir: env.GetenvLegacy(envKey, envKey),
		Disable:   cfg.Disable,
	})
	if err != nil {
		sylog.Fatalf("Failed to create an image cache handle: %s", err)
	}

	return h
}

// actionPreRun will run replaceURIWithImage and will also do the proper path unsetting
func actionPreRun(cmd *cobra.Command, args []string) {
	// For compatibility - we still set USER_PATH so it will be visible in the
	// container, and can be used there if needed. USER_PATH is not used by
	// apptainer itself in 1.0.0+
	userPath := strings.Join([]string{os.Getenv("PATH"), defaultPath}, ":")
	os.Setenv("USER_PATH", userPath)

	os.Setenv("IMAGE_ARG", args[0])

	replaceURIWithImage(cmd.Context(), cmd, args)

	// --compat infers other options that give increased OCI / Docker compatibility
	// Excludes uts/user/net namespaces as these are restrictive for many Apptainer
	// installs.
	if isCompat {
		isContainAll = true
		isWritableTmpfs = true
		noInit = true
		noUmask = true
		noEval = true
	}
}

func handleOCI(ctx context.Context, imgCache *cache.Handle, cmd *cobra.Command, pullFrom string) (string, error) {
	ociAuth, err := makeDockerCredentials(cmd)
	if err != nil {
		sylog.Fatalf("While creating Docker credentials: %v", err)
	}

	pullOpts := oci.PullOptions{
		TmpDir:     tmpDir,
		OciAuth:    ociAuth,
		DockerHost: dockerHost,
		NoHTTPS:    noHTTPS,
	}

	return oci.Pull(ctx, imgCache, pullFrom, pullOpts)
}

func handleOras(ctx context.Context, imgCache *cache.Handle, cmd *cobra.Command, pullFrom string) (string, error) {
	ociAuth, err := makeDockerCredentials(cmd)
	if err != nil {
		return "", fmt.Errorf("while creating docker credentials: %v", err)
	}
	return oras.Pull(ctx, imgCache, pullFrom, tmpDir, ociAuth, noHTTPS)
}

func handleLibrary(ctx context.Context, imgCache *cache.Handle, pullFrom string) (string, error) {
	r, err := library.NormalizeLibraryRef(pullFrom)
	if err != nil {
		return "", err
	}

	// Default "" = use current remote endpoint
	var libraryURI string
	if r.Host != "" {
		if noHTTPS {
			libraryURI = "http://" + r.Host
		} else {
			libraryURI = "https://" + r.Host
		}
	}

	c, err := getLibraryClientConfig(libraryURI)
	if err != nil {
		return "", err
	}
	return library.Pull(ctx, imgCache, r, runtime.GOARCH, tmpDir, c)
}

func handleShub(ctx context.Context, imgCache *cache.Handle, pullFrom string) (string, error) {
	return shub.Pull(ctx, imgCache, pullFrom, tmpDir, noHTTPS)
}

func handleNet(ctx context.Context, imgCache *cache.Handle, pullFrom string) (string, error) {
	return net.Pull(ctx, imgCache, pullFrom, tmpDir)
}

func replaceURIWithImage(ctx context.Context, cmd *cobra.Command, args []string) {
	// If args[0] is not transport:ref (ex. instance://...) formatted return, not a URI
	t, _ := uri.Split(args[0])
	if t == "instance" || t == "" {
		return
	}

	var image string
	var err error

	// Create a cache handle only when we know we are are using a URI
	imgCache := getCacheHandle(cache.Config{Disable: disableCache})
	if imgCache == nil {
		sylog.Fatalf("failed to create a new image cache handle")
	}

	switch t {
	case uri.Library:
		image, err = handleLibrary(ctx, imgCache, args[0])
	case uri.Oras:
		image, err = handleOras(ctx, imgCache, cmd, args[0])
	case uri.Shub:
		image, err = handleShub(ctx, imgCache, args[0])
	case oci.IsSupported(t):
		image, err = handleOCI(ctx, imgCache, cmd, args[0])
	case uri.HTTP:
		image, err = handleNet(ctx, imgCache, args[0])
	case uri.HTTPS:
		image, err = handleNet(ctx, imgCache, args[0])
	default:
		sylog.Fatalf("Unsupported transport type: %s", t)
	}

	if err != nil {
		sylog.Fatalf("Unable to handle %s uri: %v", args[0], err)
	}

	args[0] = image
}

// ExecCmd represents the exec command
var ExecCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(2),
	PreRun:                actionPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/exec"}, args[1:]...)
		if err := launchContainer(cmd, args[0], a, ""); err != nil {
			sylog.Fatalf("%s", err)
		}
	},

	Use:     docs.ExecUse,
	Short:   docs.ExecShort,
	Long:    docs.ExecLong,
	Example: docs.ExecExamples,
}

// ShellCmd represents the shell command
var ShellCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(1),
	PreRun:                actionPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			sylog.Warningf("Parameters to shell command are ignored")
		}

		a := []string{"/.singularity.d/actions/shell"}
		if err := launchContainer(cmd, args[0], a, ""); err != nil {
			sylog.Fatalf("%s", err)
		}
	},

	Use:     docs.ShellUse,
	Short:   docs.ShellShort,
	Long:    docs.ShellLong,
	Example: docs.ShellExamples,
}

// RunCmd represents the run command
var RunCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(1),
	PreRun:                actionPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/run"}, args[1:]...)
		if err := launchContainer(cmd, args[0], a, ""); err != nil {
			sylog.Fatalf("%s", err)
		}
	},

	Use:     docs.RunUse,
	Short:   docs.RunShort,
	Long:    docs.RunLong,
	Example: docs.RunExamples,
}

// TestCmd represents the test command
var TestCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Args:                  cobra.MinimumNArgs(1),
	PreRun:                actionPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		a := append([]string{"/.singularity.d/actions/test"}, args[1:]...)
		if err := launchContainer(cmd, args[0], a, ""); err != nil {
			sylog.Fatalf("%s", err)
		}
	},

	Use:     docs.RunTestUse,
	Short:   docs.RunTestShort,
	Long:    docs.RunTestLong,
	Example: docs.RunTestExample,
}

func launchContainer(cmd *cobra.Command, image string, args []string, instanceName string) error {
	ns := launcher.Namespaces{
		User: userNamespace,
		UTS:  utsNamespace,
		PID:  pidNamespace,
		IPC:  ipcNamespace,
		Net:  netNamespace,
	}

	cgJSON, err := getCgroupsJSON()
	if err != nil {
		return err
	}
	if cgJSON != "" && strings.HasPrefix(image, "instance://") {
		cgJSON = ""
		sylog.Warningf("Resource limits & cgroups configuration are only applied to instances at instance start.")
	}

	ki, err := getEncryptionMaterial(cmd)
	if err != nil {
		return err
	}

	opts := []launcher.Option{
		launcher.OptWritable(isWritable),
		launcher.OptWritableTmpfs(isWritableTmpfs),
		launcher.OptOverlayPaths(overlayPath),
		launcher.OptScratchDirs(scratchPath),
		launcher.OptWorkDir(workdirPath),
		launcher.OptHome(
			homePath,
			cmd.Flag(actionHomeFlag.Name).Changed,
			noHome,
		),
		launcher.OptMounts(bindPaths, mounts, fuseMount),
		launcher.OptNoMount(noMount),
		launcher.OptNvidia(nvidia, nvCCLI),
		launcher.OptNoNvidia(noNvidia),
		launcher.OptRocm(rocm),
		launcher.OptNoRocm(noRocm),
		launcher.OptContainLibs(containLibsPath),
		launcher.OptEnv(apptainerEnv, apptainerEnvFile, isCleanEnv),
		launcher.OptNoEval(noEval),
		launcher.OptNamespaces(ns),
		launcher.OptNetwork(network, networkArgs),
		launcher.OptHostname(hostname),
		launcher.OptDNS(dns),
		launcher.OptCaps(addCaps, dropCaps),
		launcher.OptAllowSUID(allowSUID),
		launcher.OptKeepPrivs(keepPrivs),
		launcher.OptNoPrivs(noPrivs),
		launcher.OptSecurity(security),
		launcher.OptNoUmask(noUmask),
		launcher.OptCgroupsJSON(cgJSON),
		launcher.OptConfigFile(configurationFile),
		launcher.OptShellPath(shellPath),
		launcher.OptCwdPath(cwdPath),
		launcher.OptFakeroot(isFakeroot),
		launcher.OptBoot(isBoot),
		launcher.OptNoInit(noInit),
		launcher.OptContain(isContained),
		launcher.OptContainAll(isContainAll),
		launcher.OptAppName(appName),
		launcher.OptKeyInfo(ki),
		launcher.OptCacheDisabled(disableCache),
		launcher.OptDMTCPLaunch(dmtcpLaunch),
		launcher.OptDMTCPRestart(dmtcpRestart),
		launcher.OptUnsquash(unsquash),
		launcher.OptIgnoreSubuid(ignoreSubuid),
		launcher.OptIgnoreFakerootCmd(ignoreFakerootCmd),
		launcher.OptIgnoreUserns(ignoreUserns),
		launcher.OptUseBuildConfig(useBuildConfig),
		launcher.OptTmpDir(tmpDir),
		launcher.OptUnderlay(underlay),
	}

	var l launcher.Launcher

	if ociRuntime {
		sylog.Debugf("Using OCI runtime launcher.")
		l, err = ocilauncher.NewLauncher(opts...)
		if err != nil {
			return fmt.Errorf("while configuring container: %s", err)
		}
	} else {
		sylog.Debugf("Using native runtime launcher.")
		l, err = native.NewLauncher(opts...)
		if err != nil {
			return fmt.Errorf("while configuring container: %s", err)
		}
	}

	return l.Exec(cmd.Context(), image, args, instanceName)
}
