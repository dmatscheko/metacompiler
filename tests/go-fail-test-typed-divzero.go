/* Go division of a typed float constant by a constant zero. go rejects it at
 * compile time:
 *   invalid operation: division by zero
 * so this file SHOULD fail in both halves before it prints anything -
 * although the division is never reached. */

package main

import "fmt"

func main() {
	n := 0
	fmt.Println("compiled")
	if n > 0 {
		fmt.Println(float64(1) / 0)
	}
}
