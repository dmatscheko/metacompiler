/* Go typed constant arithmetic that overflows its type. go rejects it at
 * compile time:
 *   a + a (constant 400 of type uint8) overflows uint8
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

const a uint8 = 200
const b = a + a

func main() {
	fmt.Println("compiled")
	fmt.Println(b)
}
