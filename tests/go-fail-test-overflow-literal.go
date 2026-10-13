/* Go untyped-constant overflow at a composite literal's element. go rejects it at compile time:
 *   cannot use 300 (untyped int constant) as uint8 value in struct literal (overflows)
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

type P struct {
	a int
	b uint8
}

func main() {
	fmt.Println("compiled")
	fmt.Println([]P{{1, 2}, {3, 300}})
}
