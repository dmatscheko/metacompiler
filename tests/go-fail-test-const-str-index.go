/* Go constant index beyond a NAMED constant string. go rejects it at compile
 * time:
 *   invalid argument: index 3 out of bounds [0:3]
 * so this file SHOULD fail in both halves before it prints anything -
 * although the index is never reached. */

package main

import "fmt"

const s = "abc"

func main() {
	fmt.Println("compiled")
	if len(s) > 3 {
		fmt.Println(s[3])
	}
}
