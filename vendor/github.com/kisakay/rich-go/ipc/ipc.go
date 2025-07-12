package ipc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
)

var socket net.Conn

// Choose the right directory to the ipc socket and return it
func GetIpcPath() string {
	variablesnames := []string{"XDG_RUNTIME_DIR", "TMPDIR", "TMP", "TEMP"}

	if _, err := os.Stat("/run/user/1000/snap.discord"); err == nil {
		return "/run/user/1000/snap.discord"
	}

	if _, err := os.Stat("/run/user/1000/.flatpak/com.discordapp.Discord/xdg-run"); err == nil {
		return "/run/user/1000/.flatpak/com.discordapp.Discord/xdg-run"
	}

	for _, variablename := range variablesnames {
		path, exists := os.LookupEnv(variablename)

		if exists {
			return path
		}
	}

	return "/tmp"
}

func CloseSocket() error {
	if socket != nil {
		socket.Close()
		socket = nil
	}
	return nil
}

// Read the socket response
func Read() string {
	buf := make([]byte, 512)
	payloadlength, err := socket.Read(buf)
	if err != nil {
		//fmt.Println("Nothing to read")
	}

	buffer := new(bytes.Buffer)
	for i := 8; i < payloadlength; i++ {
		buffer.WriteByte(buf[i])
	}

	return buffer.String()
}

// Send opcode and payload to the unix (or named) socket.
// If the current socket connection is broken (for instance, when Discord is
// restarted and the underlying IPC file/pipe disappears) we transparently
// attempt to reopen it once before giving up. This prevents the common "write
// unix ...: broken pipe" error the next time the library tries to communicate
// after a Discord restart.
func Send(opcode int, payload string) string {
	// Lazily (re-)open the socket if it is not available
	if socket == nil {
		if err := OpenSocket(); err != nil {
			fmt.Println("ipc: unable to establish ipc connection:", err)
			return ""
		}
	}

	// Buffer construction for the frame to send
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, int32(opcode)); err != nil {
		fmt.Println(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, int32(len(payload))); err != nil {
		fmt.Println(err)
	}
	buf.Write([]byte(payload))

	// Helper to attempt a single write on the current socket
	tryWrite := func() error {
		_, err := socket.Write(buf.Bytes())
		return err
	}

	// First write attempt
	if err := tryWrite(); err != nil {
		// The write failed – most likely because Discord restarted. Try to reopen
		// the socket once and write again.
		fmt.Println("ipc: write failed, attempting to reopen socket:", err)
		_ = CloseSocket()
		if errOpen := OpenSocket(); errOpen != nil {
			fmt.Println("ipc: reopen failed:", errOpen)
			return ""
		}
		if errRetry := tryWrite(); errRetry != nil {
			fmt.Println("ipc: write after reopen failed:", errRetry)
			return ""
		}
	}

	return Read()
}
