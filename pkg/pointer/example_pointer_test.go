package pointer_test

import (
	"fmt"
	"time"

	"github.com/mycujoo/go-stdlib/pkg/pointer"
)

type point struct {
	x int
	y int
}

func (p point) String() string {
	return fmt.Sprintf("(%d,%d)", p.x, p.y)
}

func ExampleFrom() {
	// From empty string it returns pointer to empty string
	ps0 := pointer.From("")
	ps1 := pointer.From("Hello")
	fmt.Printf("%v,%v\n", *ps0, *ps1)
	pi := pointer.From(1)
	fmt.Printf("%v\n", *pi)
	pstr := pointer.From(point{0, 5})
	fmt.Printf("%#v\n", pstr)
	// Output: ,Hello
	// 1
	// &pointer_test.point{x:0, y:5}
}

func ExampleOptional() {
	// Optional pointer of zero is nil
	pi0 := pointer.Optional(0)
	pi1 := pointer.Optional(1)
	fmt.Printf("%v %#v\n", pi0, *pi1)
	// Same applies to the string
	ps0 := pointer.Optional("")
	ps1 := pointer.Optional("1")
	fmt.Printf("%v %#v\n", ps0, *ps1)
	// Same applies to the struct
	pstr0 := pointer.Optional(point{})
	pstr1 := pointer.Optional(point{1, 2})
	fmt.Printf("%#v %#v\n", pstr0, pstr1)

	// works for time as well
	pt0 := pointer.Optional(time.Time{})
	pt1 := pointer.Optional(time.Date(2022, 5, 22, 0, 0, 0, 0, time.UTC))
	fmt.Printf("%#v %#v\n", pt0, pt1)

	// Optional won't work for interfaces
	// var any0 any
	// any0 = point{}
	// pany0 := pointer.Optional(any0)

	// Output: <nil> 1
	// <nil> "1"
	// (*pointer_test.point)(nil) &pointer_test.point{x:1, y:2}
	// <nil> time.Date(2022, time.May, 22, 0, 0, 0, 0, time.UTC)
}

func ExampleUnwrap() {
	// Unwrapping integers works as expected
	pi0 := new(int)
	pi1 := pointer.From(1)
	i0 := pointer.Unwrap(pi0)
	i1 := pointer.Unwrap(pi1)
	fmt.Printf("%v %v\n", i0, i1)

	point0 := new(point)
	point1 := &point{1, 5}
	// Unwrapping <nil> results in 0 filled struct
	pstr0 := pointer.Unwrap(point0)
	pstr1 := pointer.Unwrap(point1)
	fmt.Printf("%v %v\n", pstr0, pstr1)

	// Output: 0 1
	// (0,0) (1,5)
}
