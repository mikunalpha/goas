package main

import (
	"flag"
	"log"
)

func main() {
	modulePath := flag.String("module-path", "", "goas will search @comment under the module")
	mainFilePath := flag.String("module-file-path", "", "goas will start to search @comment from this main file")
	handlerPath := flag.String("handler-path", "", "goas only search handleFunc comments under the path")
	output := flag.String("output", "oas.json", "output file")
	debug := flag.Bool("debug", false, "show debug message")

	p, err := newParser(*modulePath, *mainFilePath, *handlerPath, *debug)
	if err != nil {
		log.Fatal(err)
	}

	if err := p.CreateOASFile(*output); err != nil {
		log.Fatal(err)
	}
}
