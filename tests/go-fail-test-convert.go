/* Go conversion of a constant its type cannot hold. go rejects it at compile time:
 *   constant 256 overflows uint8
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

func main() {
	fmt.Println("compiled")
	fmt.Println(uint8(256))
}
