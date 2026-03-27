package main

import (
	"github.com/cy77cc/OpsPilot/internal/cmd"
	"github.com/subosito/gotenv"
)

func main() {
	gotenv.Load(".env")
	cmd.Execute()
}
