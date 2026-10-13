/* Go integer division by a constant zero. go rejects it at compile time:
 *   invalid operation: division by zero
 * (a float division by a constant zero is not an error unless both operands
 * are constants), so this file SHOULD fail in both halves before it prints
 * anything - although the division is never reached. */

package main

import "fmt"

func main() {
	x := 7
	f := 2.5
	fmt.Println("compiled", f/0)
	if x > 7 {
		fmt.Println(x / 0)
	}
}
