/* Go untyped-constant overflow at a typed constant declaration. go rejects it at compile time:
 *   cannot use 256 (untyped int constant) as uint8 value in constant declaration (overflows)
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

const c uint8 = 256

func main() {
	fmt.Println("compiled")
	fmt.Println(c)
}
