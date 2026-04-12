package main

import (
	"github.com/subosito/gotenv"
)

func main() {
	gotenv.Load(".env")
	Execute()
}
