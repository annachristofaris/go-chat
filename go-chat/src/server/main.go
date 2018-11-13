package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool) // map where key is  a pointer to a WebSocket, connected clients
var broadcast = make(chan Message)           // broadcast channel that will act as a queue for sent messages

var upgrader = websocket.Upgrader{} //instance of an Upgrader

// object that will hold messages
type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

func main() {
	// Simple static fileserver
	fs := http.FileServer(http.Dir("../"))
	http.Handle("/", fs)

	//route that handles requests for initiating a WebSocket
	http.HandleFunc("/ws", handleConnections)

	// handleMessages() is a goroutine that will run concurrently, listenng for incoming messages
	go handleMessages()

	// Start server and log any errors
	log.Println("Server started on :9999")
	err := http.ListenAndServe(":9999", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// function to handle incoming WebSocket connections
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// close the connection when the function returns
	defer ws.Close() // defer lets Go know to close out our WebSocket connection when the function returns

	// register new client
	clients[ws] = true

	// inifinite loop that continously waits for a new messages to be written to the WebSocket,
	// unserializes it from JSON to a Message object and then throws it into the broadcast channel
	for {
		var msg Message
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		// Send newly-received message to the broadcast channel
		broadcast <- msg
	}
}

// function that loops continuously, receving messages from the broadcast channel and
// relays the message to clients over their WebSocket connection
func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send message to all currently-connected clients
		for client := range clients {
			err := client.WriteJSON(msg)
			// if there's is an error writing to the WebSocket, close connection and remove it from clients map
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
