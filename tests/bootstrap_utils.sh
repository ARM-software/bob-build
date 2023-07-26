#!/usr/bin/env bash




function create_link() {
    local target="$1" linkname="$2"

    if [[ ! -e "$linkname" ]]; then
        # Link doesn't exist
        :
    elif [[ -L "$linkname" ]]; then
        # Link exists. Check if it already points to the right place.
        if [[ $(readlink "$linkname") = "$target" ]]; then
            return 0 # Already refers to the correct location
        else
            rm "$linkname" || return $?
        fi
    else
        # Link location is already in use - fail rather than deleting it
        # without warning.
        echo "Can't create link from $linkname to $target - $linkname already exists and is not a symbolic link"
        return 1
    fi
    ln -s "$target" "$linkname"
}
