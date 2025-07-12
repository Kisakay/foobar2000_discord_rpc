package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/kisakay/rich-go/client"
)

const (
	clientID              = "1393213730786906273"
	nowPlayingPath        = "/Users/anais/Documents/nowplaying.txt"
	stringForClosedFoobar = "Stopped Running"
)

var isConnected = false

func readFirstLine(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	return "", scanner.Err()
}

func updateRPC(state string) {
	if state == stringForClosedFoobar {
		if isConnected {
			client.Logout()
			isConnected = false
			fmt.Println("foobar2000 is closed. RPC is hidden")
		}
		return
	}

	if !isConnected {
		err := client.Login(clientID)
		fmt.Println("foobar2000 is running. Starting the RPC")

		if err != nil {
			fmt.Println("RPC connection error: ", err)
			return
		}
		isConnected = true
	}

	err := client.SetActivity(client.Activity{
		State:      state,
		LargeImage: "foobar2000",
		LargeText:  "www.foobar2000.org",
	})

	if err != nil {
		fmt.Println("Error while the RPC update: ", err)
		// Most common when Discord has been restarted and the previous IPC socket is closed.
		// Reset the connection so that the next update triggers a fresh login.
		client.Logout()
		isConnected = false
	}
}

func main() {
	initialState, err := readFirstLine(nowPlayingPath)
	if err == nil {
		updateRPC(initialState)
	}

	fmt.Println("foobar2000_discord_rpc is running. Watching the nowplaying.txt...")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Watcher error: ", err)
		os.Exit(1)
	}
	defer watcher.Close()

	err = watcher.Add(nowPlayingPath)
	if err != nil {
		fmt.Println("Watcher add error: ", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	current := initialState

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					state, err := readFirstLine(nowPlayingPath)
					if err == nil && state != current {
						updateRPC(state)
						current = state
					}
				}
			case err := <-watcher.Errors:
				fmt.Println("Watcher error: ", err)
			}
		}
	}()

	<-sigChan
	if isConnected {
		client.Logout()
	}
	fmt.Println("\nSIGINT signal called. foobar2000_discord_rpc exiting...")
}
