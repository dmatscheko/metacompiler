// ABORT PROBE: an integer divide by zero no recover() catches. It is a run-time
// panic, so the deferred call still runs and go writes, on stderr with exit
// status 2, `panic: runtime error: integer divide by zero`. The program never
// mentions recover, so the compiled divisions carry no pre-check and natively it
// is the C floor that raises the panic. See the "SHOULD ABORT" rows in
// .vscode/launch.json.
package main

import "fmt"

func div(a, b int) int {
	defer fmt.Println("deferred before the abort")
	return a / b
}

func main() {
	zs := []int{7, 0}
	fmt.Println("before the abort")
	fmt.Println(div(zs[0], zs[1]))
}
