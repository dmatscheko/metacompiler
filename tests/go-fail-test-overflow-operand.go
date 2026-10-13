/* Go untyped-constant overflow at a binary operand beside a typed one. go rejects it at compile time:
 *   300 (untyped int constant) overflows uint8
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

func main() {
	var u8 uint8
	fmt.Println("compiled")
	fmt.Println(u8 == 300)
}
