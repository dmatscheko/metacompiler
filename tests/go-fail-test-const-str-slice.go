/* Go constant slice bound beyond a constant string built with '+'. go rejects
 * it at compile time:
 *   invalid argument: index 6 out of bounds [0:5]
 * so this file SHOULD fail in both halves before it prints anything -
 * although the slice is never reached. */

package main

import "fmt"

const s = "ab"
const t = s + "cd"

func main() {
	fmt.Println("compiled")
	if len(t) > 4 {
		fmt.Println(t[1:6])
	}
}
