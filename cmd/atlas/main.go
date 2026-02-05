package main

import (
	"log"

	"github.com/IYouKnow/atlas-drive/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
