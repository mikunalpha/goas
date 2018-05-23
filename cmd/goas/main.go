package main

import (
	"log"

	"github.com/mikunalpha/goas"
)

func main() {
	g := goas.New()
	// fmt.Printf("%+v\n", g)

	err := g.CreateOASFile("oas.json")
	if err != nil {
		log.Fatal(err)
	}
}
