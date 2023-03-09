package gendiffer

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

var (
	backendType   = flag.String("backend", "linux", "Type of backend to use")
	bobBinaryPath = flag.String("bob_binary_path", "", "Path to the bob binary to test.")
	configFile    = flag.String("config_file", "", "Path to the config file for Linux.")
	configJson    = flag.String("config_json", "", "Path to the config json file for Linux.")
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

func singleBobGenerationTest(t *testing.T, args *generationArgs) {
	shouldUpdate := os.Getenv("UPDATE_SNAPSHOTS") != ""
	setCommonEnv(t, args)

	outputFile := "build.ninja"
	outputDir := "out/linux/"
	if args.BackendType == "android" {
		outputFile = "Android.bp"
		outputDir = "out/android/"
	}
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer
	runCmd := exec.Command(args.BobBinaryPath, "-l", args.BobRootAbsolute+"/bplist", "-b", args.BobRootAbsolute, "-n", args.BobRootAbsolute, "-d", args.BobRootAbsolute+"/ninja.build.d", "-o", args.BobRootAbsolute+"/build.ninja", args.BobRootAbsolute+"/build.bp")
	runCmd.Stdout = &stdOut
	runCmd.Stderr = &stdErr
	err := runCmd.Run()
	if err != nil {
		t.Fatal("Failed executing bob with error: ", runCmd.Stderr)
	}

	// Check Files
	outFile := getFileContents(t, path.Join(args.BobRootAbsolute, outputFile))
	outFile = redactWorkspacePath(outFile, args.BobRootAbsolute)

	expectedStdout := getFileContents(t, path.Join(args.TestDataPathAbsolute, outputDir, "expectedStdout.txt"))
	expectedStderr := getFileContents(t, path.Join(args.TestDataPathAbsolute, outputDir, "expectedStderr.txt"))
	expectedFile := getFileContents(t, path.Join(args.TestDataPathAbsolute, outputDir, outputFile))

	if shouldUpdate {
		os.WriteFile(path.Join(args.SrcTestDirectory, outputDir, "expectedStdout.txt"), stdOut.Bytes(), 0644)
		os.WriteFile(path.Join(args.SrcTestDirectory, outputDir, "expectedStderr.txt"), stdErr.Bytes(), 0644)
		os.WriteFile(path.Join(args.SrcTestDirectory, outputDir, outputFile), []byte(outFile), 0644)
	} else {
		if outFile != expectedFile {
			t.Fatal(outputFile, " mismatch.")
		}
		if stdOut.String() != expectedStdout {
			t.Fatal(args.BackendType, " stdout mismatch")
		}
		if stdErr.String() != expectedStderr {
			t.Fatal(args.BackendType, " stderr mismatch.")
		}
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

func getFileContents(t *testing.T, filename string) string {
	// We wrap this call as if no file exists, we can centralize the error.
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Logf("File: ", filename)
		t.Logf("File not found. \n Please run UPDATE_SNAPSHOTS=true bazel run ", os.Getenv("TEST_TARGET"))
	}
	return string(data)
}

func redactWorkspacePath(s, wsPath string) string {
	// We must cleanup a specific Android.bp target that is unique to each generation. It is non-hermetic.
	re := regexp.MustCompile("genrule {\n.*_check_buildbp_updates.*\n.*\n.*\n.*\n.*\n.*")
	res := re.ReplaceAllString(s, "")
	return strings.ReplaceAll(res, wsPath, "%WORKSPACEPATH%")
}
