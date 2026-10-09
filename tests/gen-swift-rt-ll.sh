#!/usr/bin/env bash
# Regenerate languages/lib/swift-rt.ll; see tests/gen-rt-ll.sh.
exec "$(dirname "$0")/gen-rt-ll.sh" swift "$@"
