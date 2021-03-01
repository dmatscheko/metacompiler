package main

import (
	"fmt"
	"regexp"
	"strings"
)

type object = interface{}
type sequence = []object
type group struct{}

func jsonizeObject(ob object) string {
	pp := fmt.Sprintf("%#v", ob)
	pp = strings.ReplaceAll(pp, "[]interface {}", "")

	if strings.HasPrefix(pp, "[][]") {
		pp = strings.Replace(pp, "[][]", "[]", 1)
	}

	pp = strings.ReplaceAll(pp, "main.group{}, ", "G: ")

	space := regexp.MustCompile(`[ \t]+`)
	pp = space.ReplaceAllString(pp, " ")

	return pp
}

func appendObj(target []interface{}, elems ...interface{}) []interface{} {
	if elems == nil || len(elems) == 0 {
		return target
	}
	if elems[0] == nil {
		return target
	}

	switch elems[0].(type) {
	case group:
		if len(elems) == 1 {
			return target
		} else if len(elems) == 2 {
			return appendObj(target, elems[1])
		}
		return append(target, elems)
	}

	if len(elems) == 1 {
		if seq, ok := elems[0].(sequence); ok {
			return appendObj(target, seq...)
		}
	}

	return append(target, elems...)
}

func simplifyObj(obj interface{}, mustBeGroup ...bool /* = false */) object {
	enforceGroup := len(mustBeGroup) > 0 && mustBeGroup[0] == true

	// If the object is NOT an array, return the one element.
	o, ok := obj.([]interface{})
	if !ok {
		return obj
	}

	// If the array is empty, return nil.
	if len(o) == 0 {
		return nil
	}

	// If the object is a group, do not remove elements, even if they are nil. Still check their child arrays.
	switch o[0].(type) {
	case group:
		if len(o) == 1 { // If the object IS a group and one element, return nil, because it would be an empty group.
			return nil
		} else if len(o) == 2 { // If the object IS a group and two elements, return only the second element, because it is a single element.
			return simplifyObj(o[1], true)
		}

		res := sequence{group{}}

		for i := 1; i < len(o); i++ { // Do not drop nil elements in a group.
			// We are not allowed to delete nils, but we can split NON group objects
			res = appendObj(res, simplifyObj(o[i]))
		}
		return res
	}

	// If the object is NOT a group and one element, return the one element without the array.
	if len(o) == 1 {
		return simplifyObj(o[0], enforceGroup)
	}

	var res sequence

	if enforceGroup {
		// There was no group, so enforce one.
		res = sequence{group{}}
	} else {
		res = sequence{}
	}

	// There might have been a group, but we are in a sub array, so we are still allowed to drop nil elements.
	for i := 0; i < len(o); i++ {
		tmp := simplifyObj(o[i])
		if tmp != nil {
			res = appendObj(res, tmp)
		}
	}

	// Last cleaning for length.
	if len(res) == 0 {
		return nil
	} else if len(res) == 1 {
		if enforceGroup {
			return res
		}
		return res[0]
	} else if len(res) == 2 && enforceGroup {
		return res[1]
	}
	return res
}

func main() {
	tst1 := []interface{}{group{}, "rtu", 2, []interface{}{group{}, "OR", []interface{}{[]interface{}{group{}, "IDENT", "foo", nil, 1}, []interface{}{group{}, "TERMINAL", "B"}}}}
	simplifyObj(tst1)

	fmt.Printf("\n%s\n", jsonizeObject(simplifyObj(tst1)))
}
