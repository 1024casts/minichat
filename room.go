package main

import (
	"context"
	"fmt"
	"log"

	"github.com/1024casts/minichat/config"

	"github.com/google/uuid"
)

const welcomeMessage = "%s joined the room"

type Room struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Private    bool      `json:"private"`
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
}

// NewRoom creates a new Room
func NewRoom(name string, private bool) *Room {
	return &Room{
		ID:         uuid.New(),
		Name:       name,
		Private:    private,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
	}
}

// RunRoom runs our room, accepting various requests
func (room *Room) RunRoom() {
	// subscribe to pub/sub messages inside a new goroutine
	go room.subscribeToRoomMessages()

	for {
		select {

		case client := <-room.register:
			room.registerClientInRoom(client)

		case client := <-room.unregister:
			room.unregisterClientInRoom(client)

		case message := <-room.broadcast:
			//room.broadcastToClientsInRoom(message.encode())
			room.publishRoomMessage(message.encode())
		}
	}
}

func (room *Room) GetId() string {
	return room.ID.String()
}

func (room *Room) GetName() string {
	return room.Name
}

// Add the GetPrivate method to make Room compatible with model.Room interface
func (room *Room) GetPrivate() bool {
	return room.Private
}

func (room *Room) registerClientInRoom(client *Client) {
	// by sending the message first the new user won't see his own message.
	if !room.Private {
		room.notifyClientJoined(client)
	}
	room.clients[client] = true
}

func (room *Room) unregisterClientInRoom(client *Client) {
	if _, ok := room.clients[client]; ok {
		delete(room.clients, client)
	}
}

func (room *Room) broadcastToClientsInRoom(message []byte) {
	for client := range room.clients {
		client.send <- message
	}
}

func (room *Room) notifyClientJoined(client *Client) {
	message := &Message{
		Action:  SendMessageAction,
		Target:  room,
		Message: fmt.Sprintf(welcomeMessage, client.GetName()),
	}

	//room.broadcastToClientsInRoom(message.encode())
	room.publishRoomMessage(message.encode())
}

var ctx = context.Background()

func (room *Room) publishRoomMessage(message []byte) {
	err := config.Redis.Publish(ctx, room.GetName(), message).Err()

	if err != nil {
		log.Println(err)
	}
}

func (room *Room) subscribeToRoomMessages() {
	pubsub := config.Redis.Subscribe(ctx, room.GetName())

	ch := pubsub.Channel()

	for msg := range ch {
		room.broadcastToClientsInRoom([]byte(msg.Payload))
	}
}
