package main

import (
	"flag"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/ARM-software/bob-build/core"
	"github.com/ARM-software/bob-build/internal/utils"
)

func main() {
	// The primary builder should use the global flag set because the
	// bootstrap package registers its own flags there.
	flag.Parse()

	cpuprofile, present := os.LookupEnv("BOB_CPUPROFILE")
	if present && cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			utils.Die("%v", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	core.Main()

	memprofile, present := os.LookupEnv("BOB_MEMPROFILE")
	if present && memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			utils.Die("%v", err)
		}
		runtime.GC()
		pprof.WriteHeapProfile(f)
	}
}
