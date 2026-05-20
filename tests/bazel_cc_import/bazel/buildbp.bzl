def _quote(value):
    return "\"" + value.replace("\\", "\\\\").replace("\"", "\\\"") + "\""

def _string_list(name, values):
    if not values:
        return ""
    out = "    " + name + ": [\n"
    for value in sorted(values):
        out += "        " + _quote(value) + ",\n"
    out += "    ],\n"
    return out

def library_bp_content(name, src, includes, defines):
    content = "bob_import_cc_library {\n"
    content += "    name: " + _quote(name) + ",\n"
    if src:
        content += "    src: " + _quote(src) + ",\n"
    content += _string_list("includes", includes)
    content += _string_list("defines", defines)
    content += "    target: \"target\",\n"
    content += "}\n"
    return content
