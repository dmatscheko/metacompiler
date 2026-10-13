/* Go slice expression with constant bounds out of order. go rejects it at
 * compile time:
 *   invalid slice indices: 1 < 2
 * so this file SHOULD fail in both halves before it prints anything -
 * although the slice is never reached. */

package main

import "fmt"

func main() {
	xs := []int{1, 2, 3}
	fmt.Println("compiled")
	if len(xs) > 3 {
		fmt.Println(xs[2:1])
	}
}
