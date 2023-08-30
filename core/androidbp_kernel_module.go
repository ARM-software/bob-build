package core

import (
	"path/filepath"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/internal/utils"
)

func stringParam(optName string, optValue string) (opts []string) {
	if optValue != "" {
		opts = []string{optName, optValue}
	}
	return
}

func stringParams(optName string, optValueLists ...[]string) (opts []string) {
	for _, optValueList := range optValueLists {
		for _, optValue := range optValueList {
			opts = append(opts, optName)
			opts = append(opts, optValue)
		}
	}
	return
}

const prebuiltMake = "prebuilts/build-tools/linux-x86/bin/make"

func (g *androidBpGenerator) kernelModuleActions(ko *ModuleKernelObject, ctx blueprint.ModuleContext) {
	if !enabledAndRequired(ko) {
		return
	}

	bpmod, err := AndroidBpFile().NewModule("genrule_bob", ko.Name())
	if err != nil {
		panic(err)
	}

	outs := []string{ko.outputName() + ".ko"}
	kmod_build := getBackendPathInBobScriptsDir(g, "kmod_build.py")

	sources_param := "${in}"
	var generated_deps []string
	for _, mod := range ko.extraSymbolsModules(ctx) {
		generated_deps = append(generated_deps, mod.Name())
		// reference all dependent modules outputs, needed for related symvers files
		sources_param += " $$(dirname ${" + mod.Name() + "_out})/Module.symvers"
	}

	kdir := proptools.String(ko.Properties.Kernel_dir)
	if !filepath.IsAbs(kdir) {
		kdir = getPathInSourceDir(kdir)
	}

	addProvenanceProps(bpmod, ko)

	srcs := []string{}
	ko.Properties.GetFiles(ctx).ForEach(
		func(fp file.Path) bool {
			srcs = append(srcs, fp.UnScopedPath())
			return true
		})

	bpmod.AddStringList("srcs", srcs)
	bpmod.AddStringList("generated_deps", generated_deps)
	bpmod.AddStringList("out", outs)
	bpmod.AddStringList("implicit_outs", []string{"Module.symvers"})
	bpmod.AddStringList("tools", []string{kmod_build})
	bpmod.AddBool("depfile", true)

	// Generate the build command. Use the `stringParam` helper for options which
	// may be empty to avoid writing a flag name with no corresponding value.
	bpmod.AddStringCmd("cmd",
		[]string{
			"${tool}",
			"-o ${out}",
			"--depfile", "${depfile}",
			"--sources", sources_param,
			"--common-root", getSourceDir(),
			"--kernel", kdir,
			"--module-dir", "${gen_dir}/" + ctx.ModuleDir(),
			"--make-command", prebuiltMake,
			"--extra-cflags='" + utils.Join(ko.Properties.Cflags) + "'",
		},
		stringParam("--kbuild-options", utils.Join(ko.Properties.Kbuild_options)),
		stringParam("--cross-compile", proptools.String(ko.Properties.Kernel_cross_compile)),
		stringParam("--cc", proptools.String(ko.Properties.Kernel_cc)),
		stringParam("--hostcc", proptools.String(ko.Properties.Kernel_hostcc)),
		stringParam("--clang-triple", proptools.String(ko.Properties.Kernel_clang_triple)),
		stringParam("--ld", proptools.String(ko.Properties.Kernel_ld)),
		stringParams("-I",
			ko.Properties.Include_dirs,
			getPathsInSourceDir(ko.Properties.Local_include_dirs)),
		ko.Properties.Make_args,
	)

	addInstallProps(bpmod, ko.getInstallableProps(), ko.Properties.isProprietary())
}
