package depscan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"strings"
)

type DockerExecutor struct {
	DockerPath string
}

func NewDockerExecutor() *DockerExecutor {
	return &DockerExecutor{
		DockerPath: "docker",
	}
}

func (e *DockerExecutor) dockerPath() string {
	if e.DockerPath == "" {
		return "docker"
	}

	return e.DockerPath
}

func validateRuntime(runtime Runtime) error {
	if strings.TrimSpace(runtime.ContainerID) == "" {
		return errors.New("container ID is empty")
	}

	return nil
}

func (e *DockerExecutor) baseArgs(runtime Runtime) ([]string, error) {
	if err := validateRuntime(runtime); err != nil {
		return nil, err
	}

	args := []string{"exec"}

	if runtime.ContainerPath != "" {
		args = append(
			args,
			"--workdir",
			runtime.ContainerPath,
		)
	}

	args = append(args, runtime.ContainerID)

	return args, nil
}

func (e *DockerExecutor) Probe(
	ctx context.Context,
	runtime Runtime,
	executable string,
) error {
	if executable == "" {
		return errors.New("executable cannot be empty")
	}

	args, err := e.baseArgs(runtime)
	if err != nil {
		return err
	}

	args = append(
		args,
		"sh",
		"-c",
		`command -v "$1" >/dev/null 2>&1`,
		"depscan-probe",
		executable,
	)

	cmd := exec.CommandContext(
		ctx,
		e.dockerPath(),
		args...,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"executable %q is unavailable in container %s: %w (%s)",
			executable,
			runtime.ContainerID,
			err,
			string(bytes.TrimSpace(output)),
		)
	}

	return nil
}

func (e *DockerExecutor) Execute(
	ctx context.Context,
	runtime Runtime,
	executable string,
	args ...string,
) (CommandResult, error) {
	if executable == "" {
		return CommandResult{}, errors.New(
			"executable cannot be empty",
		)
	}

	dockerArgs, err := e.baseArgs(runtime)
	if err != nil {
		return CommandResult{}, err
	}

	dockerArgs = append(dockerArgs, executable)
	dockerArgs = append(dockerArgs, args...)

	cmd := exec.CommandContext(
		ctx,
		e.dockerPath(),
		dockerArgs...,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	result := CommandResult{
		ExitCode: -1,
	}

	runErr := cmd.Run()

	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()

	if runErr == nil {
		result.ExitCode = 0
		return result, nil
	}

	var exitErr *exec.ExitError

	if errors.As(runErr, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	}

	return result, fmt.Errorf(
		"execute %s in container %s: %w",
		executable,
		runtime.ContainerID,
		runErr,
	)
}

// resolveContainerFilePath resolves a relative source path
// inside the selected runtime container.
func resolveContainerFilePath(
	runtime Runtime,
	relativePath string,
) (string, error) {
	if err := validateRuntime(runtime); err != nil {
		return "", err
	}

	containerRoot := strings.TrimSpace(runtime.ContainerPath)

	if containerRoot == "" {
		return "", errors.New("container path is empty")
	}

	if !path.IsAbs(containerRoot) {
		return "", errors.New("container path must be absolute")
	}

	relativePath = strings.ReplaceAll(
		strings.TrimSpace(relativePath),
		"\\",
		"/",
	)

	if relativePath == "" || path.IsAbs(relativePath) {
		return "", errors.New("relative file path is invalid")
	}

	cleanPath := path.Clean(relativePath)

	if cleanPath == "." ||
		cleanPath == ".." ||
		strings.HasPrefix(cleanPath, "../") {
		return "", errors.New(
			"relative file path escapes source directory",
		)
	}

	return path.Join(
		path.Clean(containerRoot),
		cleanPath,
	), nil
}

// FileExists checks whether a regular file exists
// inside the selected runtime container.
func (e *DockerExecutor) FileExists(
	ctx context.Context,
	runtime Runtime,
	relativePath string,
) (bool, error) {
	targetPath, err := resolveContainerFilePath(
		runtime,
		relativePath,
	)
	if err != nil {
		return false, err
	}

	cmd := exec.CommandContext(
		ctx,
		e.dockerPath(),
		"exec",
		runtime.ContainerID,
		"test",
		"-f",
		targetPath,
	)

	output, err := cmd.CombinedOutput()

	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError

	if errors.As(err, &exitErr) &&
		exitErr.ExitCode() == 1 &&
		len(output) == 0 {
		return false, nil
	}

	return false, fmt.Errorf(
		"check file %q in container %s: %w (%s)",
		targetPath,
		runtime.ContainerID,
		err,
		strings.TrimSpace(string(output)),
	)
}

// ReadFile reads a file from the selected runtime container.
func (e *DockerExecutor) ReadFile(
	ctx context.Context,
	runtime Runtime,
	relativePath string,
) ([]byte, error) {
	targetPath, err := resolveContainerFilePath(
		runtime,
		relativePath,
	)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(
		ctx,
		e.dockerPath(),
		"exec",
		runtime.ContainerID,
		"cat",
		targetPath,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf(
			"read file %q from container %s: %w (%s)",
			targetPath,
			runtime.ContainerID,
			err,
			strings.TrimSpace(stderr.String()),
		)
	}

	return stdout.Bytes(), nil
}
