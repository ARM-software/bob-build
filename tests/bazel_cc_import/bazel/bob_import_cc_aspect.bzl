load("@rules_cc//cc/common:cc_info.bzl", "CcInfo")
load("//tests/bazel_cc_import/bazel:buildbp.bzl", "bp_content")
load("//tests/bazel_cc_import/bazel:name.bzl", "bp_target_name")
load("@bazel_skylib//lib:collections.bzl", "collections")
load("@bazel_skylib//lib:paths.bzl", "paths")

ImportCcAspectInfo = provider(fields = ["defines", "headers"])

def _include_dir_relative_path(header, include_dirs):
    best = None
    for include_dir in include_dirs:
        for path in [header.short_path, header.path]:
            if path != include_dir and paths.starts_with(path, include_dir):
                relative = paths.relativize(path, include_dir)
                if best == None or len(relative) < len(best):
                    best = relative
    return best

def _get_headers(compilation_info):
    include_dirs = compilation_info.system_includes.to_list() + \
                   compilation_info.includes.to_list() + \
                   compilation_info.external_includes.to_list()
    include_dirs = collections.uniq(include_dirs)

    headers = {}
    for header in compilation_info.headers.to_list():
        if headers.get(""):
            fail("More than one header dir")

        if header.is_directory:
            headers[""] = header
            continue

        include_path = _include_dir_relative_path(header, include_dirs)
        if include_path:
            headers[paths.normalize(include_path)] = header
        else:
            headers[paths.normalize(header.short_path)] = header

    return headers

def _merge_import_info(info_sets):
    headers = {}
    defines = []

    for info in info_sets:
        headers.update(info.headers)
        defines.extend(info.defines)

    return headers, collections.uniq(defines)

def _compilation_info(target, ctx):
    info_sets = []

    if CcInfo in target:
        compilation_context = target[CcInfo].compilation_context
        info_sets.append(
            ImportCcAspectInfo(
                headers = _get_headers(compilation_context),
                defines = compilation_context.defines.to_list(),
            ),
        )

    for attr_name in ["deps", "srcs"]:
        for dep in getattr(ctx.rule.attr, attr_name, []):
            if ImportCcAspectInfo in dep:
                info_sets.append(dep[ImportCcAspectInfo])

    if not info_sets:
        return {}, []

    return _merge_import_info(info_sets)

def _symlink_headers(ctx, module_dir, dir_name, headers):
    outputs = []

    for include_path in sorted(headers.keys()):
        if include_path == "":
            out = ctx.actions.declare_directory(module_dir + "/" + dir_name)
        else:
            out = ctx.actions.declare_file(module_dir + "/" + dir_name + "/" + include_path)
        ctx.actions.symlink(output = out, target_file = headers[include_path])
        outputs.append(out)

    return outputs

def _library_file(target):
    candidates = []

    def add_candidate(file):
        if file.path.endswith(".a") or file.path.endswith(".so"):
            candidates.append(file)

    # cc_shared_library has this
    for file in target[DefaultInfo].files.to_list():
        add_candidate(file)

    if not candidates and CcInfo in target:
        linking_context = target[CcInfo].linking_context
        for linker_input in linking_context.linker_inputs.to_list():
            for library in linker_input.libraries:
                for file in [
                    library.dynamic_library,
                    library.static_library,
                ]:
                    add_candidate(file)

    if len(candidates) > 1:
        fail("More than one lib output in '" + str(target) + "' " + str(candidates))

    if len(candidates) == 0:
        # header only lib
        return None

    return candidates[0]

def _symlink_library(ctx, module_dir, dir_name, library):
    include_path = paths.join(dir_name, library.basename)
    out = ctx.actions.declare_file(module_dir + "/" + include_path)
    ctx.actions.symlink(output = out, target_file = library)
    return include_path, out

def _write_bp(ctx, target_name, src, defines, includes):
    out = ctx.actions.declare_file(target_name + "/build.bp")
    ctx.actions.write(out, bp_content(target_name, src, includes, defines))
    return out

def _bob_import_cc_aspect_impl(target, ctx):
    headers, defines = _compilation_info(target, ctx)

    return [
        ImportCcAspectInfo(
            headers = headers,
            defines = defines,
        ),
    ]

bob_import_cc_aspect = aspect(
    implementation = _bob_import_cc_aspect_impl,
    attr_aspects = ["deps", "srcs"],
)

def _gen_bob_import_impl(ctx):
    output = []
    for dep in ctx.attr.deps:
        target_name = bp_target_name(dep.label)
        include_destination = "include"
        library_destination = "lib"

        outputs = []

        headers = dep[ImportCcAspectInfo].headers
        defines = dep[ImportCcAspectInfo].defines
        outputs.extend(_symlink_headers(ctx, target_name, include_destination, headers))

        includes = [include_destination]
        src = None

        library = _library_file(dep)
        if library:
            src, library_out = _symlink_library(ctx, target_name, library_destination, library)
            outputs.append(library_out)

        outputs.append(_write_bp(ctx, target_name, src, defines, includes))
        output.extend(outputs)

    return [
        DefaultInfo(
            files = depset(output),
        ),
    ]

gen_bob_import = rule(
    implementation = _gen_bob_import_impl,
    attrs = {
        "deps": attr.label_list(aspects = [bob_import_cc_aspect]),
    },
)
