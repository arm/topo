package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

func main() {
	// Ignore graceful cancellation so the test also requires forced cleanup.
	signal.Ignore(os.Interrupt)
	fmt.Println("ready")
	time.Sleep(10 * time.Second)
	fmt.Println("survived")
}
