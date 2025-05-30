package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

type Client struct {
	conn net.Conn
	name string
}

var (
	clients    = make(map[net.Conn]Client)
	messages   = make(chan string)
	join       = make(chan Client)
	disconnect = make(chan Client)
)

func main() {
	log.Println("Starting chat-server...")

	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	// start dispatcher goroutine
	go startDispatcher()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Failed to accept new connection:", err)
			continue
		}

		// handle new connection in a separate goroutine
		go handleConnection(conn)
	}
}

func startDispatcher() {
	for {
		select {
		case msg := <-messages:
			// listen on messages channel and send incoming message to all clients
			for _, client := range clients {
				_, err := client.conn.Write([]byte(msg))
				if err != nil {
					log.Println("Failed to send message to client:", err)
					continue
				}
			}
		case newClient := <-join:
			// listen on join channel and add new client to clients map
			clients[newClient.conn] = newClient
			for _, client := range clients {
				_, err := client.conn.Write([]byte(newClient.name + " joined the server!\n"))
				if err != nil {
					log.Println("Failed to send message to client:", err)
				}
			}
		case clientLeft := <-disconnect:
			// listen on disconnect channel and remove client from clients map
			delete(clients, clientLeft.conn)
			for _, client := range clients {
				_, err := client.conn.Write([]byte(clientLeft.name + " left the server!\n"))
				if err != nil {
					log.Println("Failed to send message to client:", err)
				}
			}
		}
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	// username entry logic
	var name string
	for {
		_, err := conn.Write([]byte("Enter you name: "))
		if err != nil {
			log.Println("Error writing to client:", err)
			return
		}

		name, err = reader.ReadString('\n')
		if err != nil {
			log.Println("Error reading from client:", err)
			return
		}

		name = strings.TrimSpace(name)
		if isNameTaken(name) {
			_, err := conn.Write([]byte("Username is already taken.\n"))
			if err != nil {
				log.Println("Error writing to client:", err)
				return
			}
		} else {
			break
		}
	}

	// client join
	client := Client{conn: conn, name: name}
	join <- client

	// client connected and ready to send messages
	for {
		msg, err := reader.ReadString('\n') // blocks until new message is read
		if err != nil {
			disconnect <- client
			return
		}
		messages <- fmt.Sprintf("[%s] %s", client.name, msg) // send message to message channel
	}
}

// checks if username is taken in clients map
func isNameTaken(name string) bool {
	for _, client := range clients {
		if client.name == name {
			return true
		}
	}
	return false
}
