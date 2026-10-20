/* Go untyped float constant beyond float32's range at a float32 location. go
 * rejects it at compile time:
 *   cannot use 1e40 (untyped float constant 1e+40) as float32 value in variable declaration (overflows)
 * so this file SHOULD fail in both halves before it prints anything -
 * although the declaration is never reached. */

package main

import "fmt"

func main() {
	n := 0
	fmt.Println("compiled")
	if n > 0 {
		var f float32 = 1e40
		fmt.Println(f)
	}
}
