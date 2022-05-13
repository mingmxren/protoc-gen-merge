package main

import (
	"log"

	"github.com/mingmxren/protokit"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if err := protokit.RunPlugin(NewPlugin()); err != nil {
		log.Fatal(err)
	}
}
