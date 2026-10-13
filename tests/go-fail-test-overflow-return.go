/* Go untyped-constant overflow at a returned value. go rejects it at compile time:
 *   cannot use 256 (untyped int constant) as uint8 value in return statement (overflows)
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

func f() uint8 { return 256 }

func main() {
	fmt.Println("compiled")
	fmt.Println(f())
}
