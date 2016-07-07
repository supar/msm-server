package main

import (
	"fmt"
	"os"
)

func showVersion() {
	fmt.Printf("Mail staff managment api server (%s) %s, built %s\n", NAME, VERSION, BUILDDATE)
	os.Exit(0)
}
