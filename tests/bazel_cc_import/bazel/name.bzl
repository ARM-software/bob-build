def _sanitize(value):
    sanitized = ""
    for i in range(len(value)):
        c = value[i]
        if (c >= "A" and c <= "Z") or (c >= "a" and c <= "z") or (c >= "0" and c <= "9"):
            sanitized += c
        else:
            sanitized += "_"
    return sanitized

def _strip_label_prefix(value):
    for i in range(len(value)):
        c = value[i]
        if c != "@" and c != "/":
            return value[i:]
    return ""

def bp_target_name(label):
    return _sanitize(_strip_label_prefix(str(label)))
