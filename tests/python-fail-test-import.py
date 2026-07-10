# Import resolution fail test.
# 'math' and 'collections' are resolvable stdlib modules (core.stdlibImports);
# 'notarealmodule' is an external library the runtime cannot provide.
# Default run: aborts with "unresolved import 'notarealmodule'".
# With -warn-imports: warns and runs to completion (exit 0).
# The program never uses any imported symbol, so it runs identically either way.
import math
import os.path
from collections import OrderedDict, defaultdict as dd
import notarealmodule

x = 3 + 4
if x == 7:
    print("import test ran")
    exit(0)
else:
    print("FAIL arithmetic")
    exit(1)
