package main

import (
	"fmt"
	"os"

	"graytail/cmd"
)

func main() {
	if err := cmd.GTCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
