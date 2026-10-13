/* Go shift by a negative constant. go rejects it at compile time:
 *   invalid operation: negative shift count -1 (untyped int constant)
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

func main() {
	x := 7
	fmt.Println("compiled")
	fmt.Println(x << -1)
}
