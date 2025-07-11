package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/hugolgst/rich-go/client"
)

const (
	clientID       = "1393213730786906273"
	nowPlayingPath = "/Users/anais/Documents/nowplaying.txt"
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
	if state == "not running" {
		if isConnected {
			client.Logout()
			isConnected = false
			fmt.Println("RPC désactivé (not running)")
		}
		return
	}

	if !isConnected {
		err := client.Login(clientID)
		if err != nil {
			fmt.Println("Erreur connexion RPC:", err)
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
		fmt.Println("Erreur mise à jour RPC:", err)
	}
}

func main() {
	initialState, err := readFirstLine(nowPlayingPath)
	if err == nil {
		updateRPC(initialState)
	}

	fmt.Println("RPC actif. Surveillance de nowplaying.txt...")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Erreur watcher:", err)
		os.Exit(1)
	}
	defer watcher.Close()

	err = watcher.Add(nowPlayingPath)
	if err != nil {
		fmt.Println("Erreur add watcher:", err)
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
				fmt.Println("Erreur watcher:", err)
			}
		}
	}()

	<-sigChan
	if isConnected {
		client.Logout()
	}
	fmt.Println("\nFermeture de l'application...")
}
