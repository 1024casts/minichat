package main

import (
	"encoding/json"
	"log"

	"github.com/1024casts/minichat/models"
)

const SendMessageAction = "send-message"
const JoinRoomAction = "join-room"
const LeaveRoomAction = "leave-room"
const JoinRoomPrivateAction = "join-room-private"
const RoomJoinedAction = "room-joined"

const UserJoinedAction = "user-join"
const UserLeftAction = "user-left"

type Message struct {
	Action  string      `json:"action"`
	Message string      `json:"message"`
	Target  *Room       `json:"target"`
	Sender  models.User `json:"sender"` // Use model.User interface
}

func (message *Message) encode() []byte {
	data, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
	}

	return data
}

// UnmarshalJSON custom unmarshel to create a Client instance for Sender
func (message *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	msg := &struct {
		Sender Client `json:"sender"`
		*Alias
	}{
		Alias: (*Alias)(message),
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	message.Sender = &msg.Sender
	return nil
}
