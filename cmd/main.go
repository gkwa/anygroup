package main

import (
	"os"

	"github.com/taylormonacelli/anygroup"
)

func main() {
	code := anygroup.Execute()
	os.Exit(code)
}
