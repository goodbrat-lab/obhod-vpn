package main

import (
	"bytes"
	"io/ioutil"
	"log"
)

func main() {
	b, err := ioutil.ReadFile("backend/internal/service/manager.go")
	if err != nil {
		log.Fatal(err)
	}
	
	// Remove null bytes
	b = bytes.ReplaceAll(b, []byte{0}, []byte{})
	
	err = ioutil.WriteFile("backend/internal/service/manager.go", b, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
