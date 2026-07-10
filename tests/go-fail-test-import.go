/* Go unresolvable import. "fmt" and "os" resolve against the runtime's stdlib set,
 * but "github.com/example/widget" is an external package the runtime does not
 * provide, so by default the compile stops with a clean "unresolved import" error
 * (this file SHOULD fail without flags). With -warn-imports the external import is
 * warned and ignored; the program never uses it, so it runs and exits 0. */

package main

import (
	"fmt"
	"os"
	"github.com/example/widget"
)

func main() {
	fmt.Println("ran without the widget")
	os.Exit(0)
}
