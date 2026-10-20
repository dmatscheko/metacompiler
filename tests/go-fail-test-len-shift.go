/* len of a constant string is a constant of type int, so shifting it past 63
 * bits overflows int. go rejects it at compile time:
 *   len("ab") << 70 (constant 2361183241434822606848 of type int) overflows int
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

func main() {
	fmt.Println("compiled")
	fmt.Println(len("ab") << 70)
}
