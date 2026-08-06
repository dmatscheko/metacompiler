// See mod.kt. Same loop, so the languages are comparable to each other.
package main

import "fmt"

func main() {
	s := 0
	i := 0
	for i < 40000 {
		s = s + i%7
		i = i + 1
	}
	fmt.Println(s)
}
