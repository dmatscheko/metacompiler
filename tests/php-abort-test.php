<?php
// ABORT PROBE. This program is SUPPOSED to die, and its diagnostic text is the
// thing under test. See the "SHOULD ABORT" rows in .vscode/launch.json and
// abort_norm() in test.sh.
echo "before the abort\n";
throw new Exception("abort probe");
echo "never reached\n";
