/* Go constant index out of range. go rejects it at compile time:
 *   invalid argument: index 3 out of bounds [0:3]
 * so this file SHOULD fail in both halves before it prints anything -
 * although the index is never reached. */

package main

import "fmt"

func main() {
	var a [3]int
	fmt.Println("compiled")
	if len(a) > 3 {
		fmt.Println(a[3])
	}
}
