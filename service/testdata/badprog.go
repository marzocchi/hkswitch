package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		for s := range ch {
			fmt.Printf("ignoring signal: %s\n", s)
		}
	}()

	fmt.Printf("bad-prog started with PID %d\n", os.Getpid())
	wg.Wait()
}
