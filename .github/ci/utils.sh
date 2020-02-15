#!/usr/bin/env bash

function retry() {
    local attempts=$1
    if [[ "$attempts" =~ ^-?[0-9]+$ ]]; then
        shift
    else
        attempts=3
    fi
    local rc
    for (( trial = 1; trial == 1 || trial <= attempts + 1; trial += 1 )); do
        if "$@"; then
            return 0
        else
            rc=$?
            echo "Trial $trial exited [$rc]."
            if (( trial != attempts + 1 )); then
                echo "Retrying... $*"
            fi
        fi
    done
    return $rc
}
