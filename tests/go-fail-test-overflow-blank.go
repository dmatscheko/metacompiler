/* Go untyped-constant overflow at an assignment to the blank identifier. go rejects it at compile time:
 *   cannot use 1 << 63 (untyped int constant 9223372036854775808) as int value in assignment to _ identifier (overflows)
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

func main() {
	fmt.Println("compiled")
	_ = 1 << 63
}
