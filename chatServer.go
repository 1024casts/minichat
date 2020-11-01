package main

import (
	"encoding/json"
	"log"

	"github.com/1024casts/minichat/config"
	"github.com/1024casts/minichat/models"
	"github.com/google/uuid"
)

type WsServer struct {
	// clients 连接wsServer的客户端
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	// broadcast 需要广播到客户端的数据
	broadcast chan []byte
	rooms     map[*Room]bool

	users          []models.User
	roomRepository models.RoomRepository
	userRepository models.UserRepository
}

// NewWebsocketServer creates a new WsServer type
func NewWebsocketServer(roomRepository models.RoomRepository, userRepository models.UserRepository) *WsServer {
	wsServer := &WsServer{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		rooms:      make(map[*Room]bool),

		roomRepository: roomRepository,
		userRepository: userRepository,
	}

	// Add users from database to server
	wsServer.users = userRepository.GetAllUsers()

	return wsServer
}

// Run our websocket server, accepting various requests
func (server *WsServer) Run() {
	go server.listenPubSubChannel()

	for {
		select {

		case client := <-server.register:
			server.registerClient(client)

		case client := <-server.unregister:
			server.unregisterClient(client)

		case message := <-server.broadcast:
			server.broadcastToClients(message)
		}
	}
}

func (server *WsServer) registerClient(client *Client) {
	// NEW: Add user to the repo
	server.userRepository.AddUser(client)

	//server.notifyClientJoined(client)
	// Publish user in PubSub
	server.publishClientJoined(client)

	server.listOnlineClients(client)
	server.clients[client] = true

	// NEW: Add user to the user slice
	//server.users = append(server.users, message.Sender)
}

func (server *WsServer) unregisterClient(client *Client) {
	if _, ok := server.clients[client]; ok {
		delete(server.clients, client)
		//server.notifyClientLeft(client)

		// NEW: Remove user from slice
		//for i, user := range server.users {
		//	if user.GetId() == message.Sender.GetId() {
		//		server.users[i] = server.users[len(server.users)-1]
		//		server.users = server.users[:len(server.users)-1]
		//	}
		//}

		// NEW: Remove user from repo
		server.userRepository.RemoveUser(client)

		// Publish user left in PubSub
		server.publishClientLeft(client)
	}
}

func (server *WsServer) broadcastToClients(message []byte) {
	for client := range server.clients {
		client.send <- message
	}
}

func (server *WsServer) findRoomByName(name string) *Room {
	var foundRoom *Room
	for room := range server.rooms {
		if room.GetName() == name {
			foundRoom = room
			break
		}
	}

	// NEW: if there is no room, try to create it from the repo
	if foundRoom == nil {
		// Try to run the room from the repository, if it is found.
		foundRoom = server.runRoomFromRepository(name)
	}

	return foundRoom
}

// NEW: Try to find a room in the repo, if found Run it.
func (server *WsServer) runRoomFromRepository(name string) *Room {
	var room *Room
	dbRoom := server.roomRepository.FindRoomByName(name)
	if dbRoom != nil {
		room = NewRoom(dbRoom.GetName(), dbRoom.GetPrivate())
		room.ID, _ = uuid.Parse(dbRoom.GetId())

		go room.RunRoom()
		server.rooms[room] = true
	}

	return room
}

func (server *WsServer) createRoom(name string, private bool) *Room {
	room := NewRoom(name, private)
	go room.RunRoom()
	server.rooms[room] = true

	// NEW: Add room to repo
	server.roomRepository.AddRoom(room)

	return room
}

func (server *WsServer) notifyClientJoined(client *Client) {
	message := &Message{
		Action: UserJoinedAction,
		Sender: client,
	}

	server.broadcastToClients(message.encode())
}

func (server *WsServer) notifyClientLeft(client *Client) {
	message := &Message{
		Action: UserLeftAction,
		Sender: client,
	}

	server.broadcastToClients(message.encode())
}

func (server *WsServer) listOnlineClients(client *Client) {
	// NEW: Use the users slice instead of the client map
	for _, user := range server.users {
		message := &Message{
			Action: UserJoinedAction,
			Sender: user,
		}
		client.send <- message.encode()
	}
}

func (server *WsServer) findRoomByID(ID string) *Room {
	var foundRoom *Room
	for room := range server.rooms {
		if room.GetId() == ID {
			foundRoom = room
			break
		}
	}

	return foundRoom
}

func (server *WsServer) findClientByID(ID string) *Client {
	var foundClient *Client
	for client := range server.clients {
		if client.ID.String() == ID {
			foundClient = client
			break
		}
	}

	return foundClient
}

const PubSubGeneralChannel = "general"

// Publish userJoined message in pub/sub
func (server *WsServer) publishClientJoined(client *Client) {

	message := &Message{
		Action: UserJoinedAction,
		Sender: client,
	}

	if err := config.Redis.Publish(ctx, PubSubGeneralChannel, message.encode()).Err(); err != nil {
		log.Println(err)
	}
}

// Publish userleft message in pub/sub
func (server *WsServer) publishClientLeft(client *Client) {

	message := &Message{
		Action: UserLeftAction,
		Sender: client,
	}

	if err := config.Redis.Publish(ctx, PubSubGeneralChannel, message.encode()).Err(); err != nil {
		log.Println(err)
	}
}

// Listen to pub/sub general channels
func (server *WsServer) listenPubSubChannel() {

	pubsub := config.Redis.Subscribe(ctx, PubSubGeneralChannel)
	ch := pubsub.Channel()
	for msg := range ch {

		var message Message
		if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
			log.Printf("Error on unmarshal JSON message %s", err)
			return
		}

		switch message.Action {
		case UserJoinedAction:
			server.handleUserJoined(message)
		case UserLeftAction:
			server.handleUserLeft(message)
		case JoinRoomPrivateAction:
			server.handleUserJoinPrivate(message)
		}
	}
}

func (server *WsServer) handleUserJoined(message Message) {
	// Add the user to the slice
	server.users = append(server.users, message.Sender)
	server.broadcastToClients(message.encode())
}

func (server *WsServer) handleUserLeft(message Message) {
	// Remove the user from the slice
	for i, user := range server.users {
		if user.GetId() == message.Sender.GetId() {
			server.users[i] = server.users[len(server.users)-1]
			server.users = server.users[:len(server.users)-1]
		}
	}
	server.broadcastToClients(message.encode())
}

func (server *WsServer) handleUserJoinPrivate(message Message) {
	// Find client for given user, if found add the user to the room.
	targetClient := server.findClientByID(message.Message)
	if targetClient != nil {
		targetClient.joinRoom(message.Target.GetName(), message.Sender)
	}
}

// Add the findUserByID method used by client.go
func (server *WsServer) findUserByID(ID string) models.User {
	var foundUser models.User
	for _, client := range server.users {
		if client.GetId() == ID {
			foundUser = client
			break
		}
	}

	return foundUser
}
