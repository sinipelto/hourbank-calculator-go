package main

import (
	"fmt"
)

type Growler interface {
	Growl() bool
}

type Cat struct {
	Name string
	Age  int
}

// *Cat is good for both objects and "references" (pointers to objects)
func (c *Cat) Speak() bool {
	fmt.Println("Meow!")
	return true
}

func (c *Cat) Growl() bool {
	fmt.Println("Grrr!")
	return true
}

func main() {
	var felix Cat // is not a pointer
	_ = felix
	// felix.Speak() // works :-)
	// felix.Growl() // works :-)

	var ginger *Cat = new(Cat)
	_ = ginger
	// ginger.Speak() // works :-)
	// ginger.Growl() // works :-)

	var g Growler = ginger
	g.Growl()

	var h Growler = &felix
	h.Growl()
}
