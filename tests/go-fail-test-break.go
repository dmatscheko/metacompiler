/* Go break outside a loop, switch or select. go rejects it at compile time:
 *   break is not in a loop, switch, or select
 * The break in the function literal does not leave the loop around it, so this
 * file SHOULD fail in both halves before it prints anything. */

package main

import "fmt"

func main() {
	fmt.Println("compiled")
	for i := 0; i < 2; i++ {
		func() {
			break
		}()
	}
}
