/* Go untyped-constant overflow. An untyped constant reaching an interface takes
 * its default type, int, and go rejects one that int cannot represent at compile
 * time:
 *   cannot use math.MaxUint64 (untyped int constant 18446744073709551615) as int
 *   value in argument to fmt.Println (overflows)
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import (
	"fmt"
	"math"
)

func main() {
	fmt.Println("compiled")
	fmt.Println(math.MaxUint64)
}
