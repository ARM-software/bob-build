def _sanitize(value):
    sanitized = ""
    for i in range(len(value)):
        c = value[i]
        if (c >= "A" and c <= "Z") or (c >= "a" and c <= "z") or (c >= "0" and c <= "9"):
            sanitized += c
        else:
            sanitized += "_"
    return sanitized

def bp_target_name(label):
    # TODO There could be collisions between targets in different repos.
    return _sanitize("%s_%s" % (label.package, label.name))
