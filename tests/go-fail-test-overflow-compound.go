/* Go untyped-constant overflow at a compound assignment's constant operand. go rejects it at compile time:
 *   256 (untyped int constant) overflows uint8
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

func main() {
	var u8 uint8
	fmt.Println("compiled")
	u8 += 256
	fmt.Println(u8)
}
