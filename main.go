package main

import (
	"fmt"
	"os"
)

func main() {
	err := RepackImage("docker-daemon:lambda-php:latest")
	if err != nil {
		fmt.Printf("Error: %+v", err)
		os.Exit(1)
	}
}
