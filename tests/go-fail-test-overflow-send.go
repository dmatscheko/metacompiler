/* Go untyped-constant overflow at a channel send. go rejects it at compile time:
 *   cannot use 256 (untyped int constant) as uint8 value in send (overflows)
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

func main() {
	ch := make(chan uint8, 1)
	fmt.Println("compiled")
	ch <- 256
	fmt.Println(<-ch)
}
