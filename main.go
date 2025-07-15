package main

import (
	"bufio"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/altfoxie/drpc"
	"github.com/fsnotify/fsnotify"
)

const (
	clientID              = "1393213730786906273"
	nowPlayingPath        = "/Users/anais/Documents/nowplaying.txt"
	stringForClosedFoobar = "Stopped Running"
)

var (
	client      *drpc.Client
	isConnected = false
)

func readAllFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func updateRPC(state []string) {
	if state[0] == stringForClosedFoobar {
		if isConnected {
			client.Close()
			isConnected = false
			println("foobar2000 is closed. RPC is hidden")
		}
		return
	}

	if !isConnected {
		println("foobar2000 is running. Starting the RPC")
		err := client.Connect()
		if err != nil {
			println("RPC connection error:", err)
			return
		}
		isConnected = true
	}

	if len(state) == 4 {
		err := client.SetActivity(drpc.Activity{
			Details: state[1] + ": " + state[2],
			State:   state[3],
			Assets: &drpc.Assets{
				LargeImage: "foobar2000",
				LargeText:  "www.foobar2000.org",
				SmallImage: chooseSmallImageWithStatus(state[0]),
				SmallText:  chooseSmallImageWithStatus(state[0]),
			},
		})

		if err != nil {
			println("Error while updating the RPC:", err)
		}
	}
}

func main() {
	var err error
	client, err = drpc.New(clientID)
	if err != nil {
		println("RPC connection error: ", err)
		os.Exit(1)
	}

	initialState, err := readAllFile(nowPlayingPath)
	if err == nil {
		updateRPC(initialState)
	}

	println("foobar2000_discord_rpc is running. Watching the nowplaying.txt...")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		println("Watcher error: ", err)
		os.Exit(1)
	}
	defer watcher.Close()

	err = watcher.Add(nowPlayingPath)
	if err != nil {
		println("Watcher add error: ", err)
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
					state, err := readAllFile(nowPlayingPath)
					if err == nil && !slicesEqual(state, current) {
						updateRPC(state)
						current = state
					}
				}
			case err := <-watcher.Errors:
				println("Watcher error: ", err)
			}
		}
	}()

	<-sigChan
	if isConnected {
		client.Close()
	}
	println("\nSIGINT signal called. foobar2000_discord_rpc exiting...")
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func chooseSmallImageWithStatus(state string) string {
	if state == "Playing" {
		return "playing"
	} else {
		return "pause"
	}
}
