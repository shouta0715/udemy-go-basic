package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	file, err := os.Create("test.txt")

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	errorLogger := log.New(io.MultiWriter(file, os.Stderr), "ERROR: ", log.LstdFlags)

	ctx, cancel := context.WithTimeout(context.Background(), 5100*time.Millisecond)
	defer cancel()

	const wdtTimeout = 800 * time.Millisecond
	const beatInterval = 500 * time.Millisecond

	heartbeat, v := task(ctx, beatInterval)

loop:
	for {
		select {
		case _, ok := <-heartbeat:

			if !ok {
				break loop
			}

			fmt.Println("beat pulse")

		case r, ok := <-v:

			if !ok {
				break loop
			}

			t := strings.Split(r.String(), "m=")
			fmt.Printf("value: %s [s]\n", t[1])
		case <-time.After(wdtTimeout):
			errorLogger.Println("do task goroutine's heartbeat stopped")
			break loop
		}
	}

}

func task(ctx context.Context, beatInterval time.Duration) (<-chan struct{}, <-chan time.Time) {
	heartBeat := make(chan struct{})
	out := make(chan time.Time)

	go func() {
		defer close(heartBeat)
		defer close(out)

		pulse := time.NewTicker(beatInterval)
		task := time.NewTicker(beatInterval * 2)

		sendPulse := func() {
			select {
			case heartBeat <- struct{}{}:
			default:
			}
		}

		sendValue := func(t time.Time) {
			for {
				select {
				case <-ctx.Done():
					return

				case <-pulse.C:
					sendPulse()

				case out <- t:
					return
				}
			}
		}

		for {
			select {
			case <-ctx.Done():
				return

			case <-pulse.C:
				sendPulse()

			case t := <-task.C:
				sendValue(t)
			}
		}
	}()

	return heartBeat, out

}
