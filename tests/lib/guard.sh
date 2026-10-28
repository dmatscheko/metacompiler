# guard.sh - sourced by every script that runs a program under test.
#
# A program under test can loop, and a looping program that prints fills
# whatever its stdout points at. When that is a file in a temp dir and the
# script dies, the EXIT trap removes the dir but the program keeps writing into
# a file that no longer has a name: du cannot see the space and it only comes
# back when the process dies. So:
#
#   1. Every file the script or anything it starts writes is capped
#      (ulimit -f, MEC_MAX_OUTPUT_MB, default 2048). The limit is inherited, so
#      it still holds for a process orphaned by a SIGKILLed script, which is the
#      one case no trap can handle. A writer over the cap dies on SIGXFSZ.
#   2. guard_reap kills every process still descending from the script. The
#      script's EXIT trap calls it before removing its temp files, and
#      HUP/INT/TERM are turned into an exit so that the trap runs.
#   3. guard_timeout SECONDS CMD... runs CMD under a wall-clock limit, for the
#      scripts that run a built binary and had none.
#
# Usage, after the script has cd'd to the repository root:
#   . tests/lib/guard.sh
#   trap 'guard_reap; rm -rf "$WORK"' EXIT

GUARD_PID=$BASHPID

# Soft limit only: a script may still raise it for one command (probe.sh's
# --max-output does).
MEC_MAX_OUTPUT_MB="${MEC_MAX_OUTPUT_MB:-2048}"
ulimit -S -f $((MEC_MAX_OUTPUT_MB * 1024))

# Stop the whole tree first, top down, so that nothing in it can start a new
# child while the rest is being killed; then kill it. Only the script itself
# reaps: a subshell that somehow ran this would otherwise kill its siblings.
guard_reap() {
    [ "$BASHPID" = "$GUARD_PID" ] || return 0
    local all=() frontier=("$GUARD_PID") next p
    while [ ${#frontier[@]} -gt 0 ]; do
        next=()
        for p in "${frontier[@]}"; do
            for p in $(pgrep -P "$p"); do
                kill -STOP "$p" 2>/dev/null
                all+=("$p"); next+=("$p")
            done
        done
        frontier=(${next[@]+"${next[@]}"})
    done
    [ ${#all[@]} -gt 0 ] && kill -9 "${all[@]}" 2>/dev/null
    return 0
}

trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

if command -v timeout >/dev/null 2>&1;    then guard_timeout() { timeout "$@"; }
elif command -v gtimeout >/dev/null 2>&1; then guard_timeout() { gtimeout "$@"; }
else guard_timeout() { perl -e 'my $t=shift; alarm $t; exec @ARGV or exit 127' "$@"; }
fi
