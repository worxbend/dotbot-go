package main

import (
	"os"

	"dotbot-go/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
