/* Go float constant with a fraction at an integer location. go rejects it at
 * compile time:
 *   cannot use 2.5 (untyped float constant) as int value in argument to f (truncated)
 * so this file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

func f(x int) int { return x }

func main() {
	fmt.Println("compiled")
	fmt.Println(f(2.5))
}
