package gendiffer

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

var (
	backendType   = flag.String("backend", "linux", "Type of backend to use")
	bobBinaryPath = flag.String("bob_binary_path", "", "Path to the bob binary to test.")
	configFile    = flag.String("config_file", "", "Path to the config file for Linux.")
	configJson    = flag.String("config_json", "", "Path to the config json file for Linux.")

	expectedStdoutFilename   = "expectedStdout.txt"
	expectedStderrFilename   = "expectedStderr.txt"
	expectedExitCodeFilename = "expectedExitCode.int"
)

type generationArgs struct {
	Name                    string
	BackendType             string
	TestDataPathAbsolute    string
	TestDataPathRelative    string
	BobRootAbsolute         string
	BobBinaryPath           string
	ConfigFile              string
	ConfigJson              string
	BuildWorkspaceDirectory string
	SrcTestDirectory        string
	ShouldUpdate            bool
}

func setCommonEnv(t *testing.T, args *generationArgs) {
	os.Setenv("TOPNAME", "build.bp")
	os.Setenv("SRCDIR", args.BobRootAbsolute)
	os.Setenv("BUILDDIR", args.BobRootAbsolute)
	os.Setenv("WORKDIR", args.BobRootAbsolute)
	os.Setenv("BOB_DIR", args.BobRootAbsolute)
	os.Setenv("BOB_LOG_WARNINGS_FILE", args.TestDataPathAbsolute+"bob_warnings.csv")
	os.Setenv("CONFIG_FILE", args.ConfigFile)
	os.Setenv("CONFIG_JSON", args.ConfigJson)
	os.Setenv("BOB_LINK_PARALLELISM", "1")
}

func diff(filename string, a []byte, b []byte) ([]byte, error) {
	tmp, err := os.MkdirTemp(os.TempDir(), "diff-")
	if err != nil {
		return nil, err
	}

	left := path.Join(tmp, "a")
	right := path.Join(tmp, "b")

	if err := os.Mkdir(left, os.ModePerm); err != nil {
		return nil, err
	}

	if err := os.WriteFile(path.Join(left, filename), a, 0644); err != nil {
		return nil, err
	}

	if err := os.Mkdir(right, os.ModePerm); err != nil {
		return nil, err
	}

	if err := os.WriteFile(path.Join(right, filename), b, 0644); err != nil {
		return nil, err
	}

	cmd := exec.Command("diff", "-bur", "--color=always", "a", "b")
	cmd.Dir = tmp
	data, err := cmd.Output()
	if len(data) > 0 {
		err = nil // ignore error code if we get a diff
	}

	return data, err
}

type DiffError struct {
	filepath string
	diff     []byte
}

func (e *DiffError) Error() string {
	return fmt.Sprintf("Difference in expected output %s:\n%s", e.filepath, e.diff)
}

func split(data []byte, sep string) ([]byte, []byte) {
	slices := bytes.SplitN(data, []byte(sep), 2)
	return slices[0], slices[1]
}

func redact(args *generationArgs, data []byte) []byte {
	redacted := []byte("${1}redacted")
	data = regexp.MustCompile(`(_check_buildbp_updates_)[a-f0-9]{10}`).ReplaceAll(data, redacted)
	data = regexp.MustCompile(`(--hash )[a-f0-9]{40}`).ReplaceAll(data, redacted)
	data = regexp.MustCompile(args.BobRootAbsolute).ReplaceAll(data, []byte("redacted"))
	return data
}

func checkFileContents(args *generationArgs, filename string, data []byte) error {
	absolute := path.Join(args.TestDataPathAbsolute, "out", args.BackendType, filename)
	relative := path.Join(args.TestDataPathRelative, "out", args.BackendType, filename)

	data = redact(args, data)

	if args.ShouldUpdate {
		if err := os.WriteFile(absolute, data, 0644); err != nil {
			return err
		}
	}

	actual, err := os.ReadFile(absolute)
	if err != nil {
		return err
	}

	if bytes.Equal(actual, data) {
		return nil
	}

	patch, err := diff(filename, actual, data)
	if err != nil {
		return err
	}

	_, suffix := split(patch, "\n")
	return &DiffError{relative, suffix}
}

func checkFile(args *generationArgs, filename string) error {
	absolute := path.Join(args.BobRootAbsolute, filename)
	data, err := os.ReadFile(absolute)
	if err != nil {
		return err
	}
	return checkFileContents(args, filename+".out", data)
}

var generated = map[string]string{
	"android": "Android.bp",
	"linux":   "build.ninja",
}

func singleBobGenerationTest(t *testing.T, args *generationArgs) {
	setCommonEnv(t, args)

	expectedExitCode := getFileInt(t, path.Join(args.TestDataPathAbsolute, "out", args.BackendType, expectedExitCodeFilename))
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer
	runCmd := exec.Command(args.BobBinaryPath, "-l", args.BobRootAbsolute+"/bplist", "-b", args.BobRootAbsolute, "-n", args.BobRootAbsolute, "-d", args.BobRootAbsolute+"/ninja.build.d", "-o", args.BobRootAbsolute+"/build.ninja", args.BobRootAbsolute+"/build.bp")
	runCmd.Stdout = &stdOut
	runCmd.Stderr = &stdErr
	err := runCmd.Run()

	if args.ShouldUpdate {
		destFile := path.Join(args.BuildWorkspaceDirectory, args.TestDataPathRelative, "out", args.BackendType, expectedExitCodeFilename)

		exitCode := 0
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}

		if _, fErr := os.Stat(destFile); errors.Is(fErr, os.ErrNotExist) {
			data := []byte(fmt.Sprintf("%d\n", exitCode))

			if err := os.WriteFile(destFile, data, 0644); err != nil {
				t.Fatalf("Cannot create file: '%s'", destFile)
			}
		}

		expectedExitCode = exitCode
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != expectedExitCode {
			t.Fatalf("Failed executing bob with error: %v, stdout: '%s', stderr: '%s'", exitErr, runCmd.Stdout, runCmd.Stderr)
		}
	} else if err != nil {
		t.Fatalf("Failed executing bob with error: %v, stdout: '%s', stderr: '%s'", err, runCmd.Stdout, runCmd.Stderr)
	} else if 0 != expectedExitCode {
		t.Fatalf("Success executing bob when expecting error: %v, stdout: '%s', stderr: '%s'", expectedExitCode, runCmd.Stdout, runCmd.Stderr)
	}

	// Check files
	errs := []error{}
	if err := checkFileContents(args, expectedStdoutFilename, stdOut.Bytes()); err != nil {
		errs = append(errs, err)
	}
	if err := checkFileContents(args, expectedStderrFilename, stdErr.Bytes()); err != nil {
		errs = append(errs, err)
	}
	if err := checkFile(args, generated[args.BackendType]); err != nil {
		if expectedExitCode == 0 || !os.IsNotExist(err) {
			errs = append(errs, err)
		}
	}
	for _, err := range errs {
		t.Log(err)
	}
	if 0 != len(errs) {
		target := os.Getenv("TEST_TARGET")
		t.Fatalf("Expected outputs did not match.\n\nRun UPDATE_SNAPSHOTS=true bazel run %s", target)
	}
}

func TestFullGeneration(t *testing.T) {
	tests := []*generationArgs{}
	runfiles, err := bazel.ListRunfiles()
	if err != nil {
		t.Fatalf("bazel.ListRunfiles() error: %v", err)
	}

	absoluteBobBinary, err := bazel.Runfile(*bobBinaryPath)
	absoluteConfigFile, err := bazel.Runfile(*configFile)
	absoluteConfigJson, err := bazel.Runfile(*configJson)

	if err != nil {
		t.Fatalf("Could not convert bob binary path %s to absolute path. Error: %v", *bobBinaryPath, err)
	}
	for _, f := range runfiles {
		// Look through runfiles for WORKSPACE files. Each WORKSPACE is a test case.
		if filepath.Base(f.Path) == "WORKSPACE" {
			// absolutePathToTestDirectory is the absolute
			// path to the test case directory. For example, /home/<user>/wksp/path/to/test_data/my_test_case
			absolutePathToTestDirectory := filepath.Dir(f.Path)
			// relativePathToTestDirectory is the workspace relative path
			// to this test case directory. For example, path/to/test_data/my_test_case
			relativePathToTestDirectory := filepath.Dir(f.ShortPath)
			// name is the name of the test directory. For example, my_test_case.
			// The name of the directory doubles as the name of the test.
			name := filepath.Base(absolutePathToTestDirectory)

			tests = append(tests, &generationArgs{
				Name:                    name,
				BackendType:             *backendType,
				TestDataPathAbsolute:    absolutePathToTestDirectory,
				TestDataPathRelative:    relativePathToTestDirectory,
				BobRootAbsolute:         absolutePathToTestDirectory + "/app",
				BobBinaryPath:           absoluteBobBinary,
				ConfigFile:              absoluteConfigFile,
				ConfigJson:              absoluteConfigJson,
				BuildWorkspaceDirectory: os.Getenv("BUILD_WORKSPACE_DIRECTORY"),
				SrcTestDirectory:        path.Join(os.Getenv("BUILD_WORKSPACE_DIRECTORY"), path.Dir(relativePathToTestDirectory), name),
				ShouldUpdate:            os.Getenv("UPDATE_SNAPSHOTS") == "true",
			})
		}
	}
	if len(tests) == 0 {
		t.Fatal("no tests found")
	}

	for _, args := range tests {
		singleBobGenerationTest(t, args)
	}
}

func getFileInt(t *testing.T, filename string) int {
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return 0
	} else if err != nil {
		t.Fatalf("Failed to read integer value %s: %v", filename, err)
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("Failed to convert integer value %s: %v", filename, err)
	}
	return value
}
