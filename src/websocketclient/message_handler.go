package websocketclient

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ChopstickLeg/Discord-Bot-Practice/src/database"
	"github.com/ChopstickLeg/Discord-Bot-Practice/src/silence"
	"github.com/ChopstickLeg/Discord-Bot-Practice/src/structs"
	"golang.org/x/exp/rand"
)

var intents int = 1<<0 | 1<<1 | 1<<9 | 1<<15

func (ws *WebsocketClient) HandleMessage(message []byte) {
	var msg structs.Message
	err := json.Unmarshal(message, &msg)
	if err != nil {
		fmt.Println("Error unmarshalling message")
	}
	switch msg.Op {
	case 10:
		//Hello message
		var data structs.HelloMessageData
		rawData, ok := msg.D.(map[string]interface{})
		if !ok {
			fmt.Println("Error asserting baseMsg.D to map[string]interface{}")
			return
		}
		rawDataBytes, err := json.Marshal(rawData)
		if err != nil {
			fmt.Println("Error marshalling rawData to bytes: ", err)
			return
		}
		err = json.Unmarshal(rawDataBytes, &data)
		if err != nil {
			fmt.Println("Error unmarshalling message: ", err)
			fmt.Println("Message: ", message)
			return
		}
		ws.HeartbeatInterval = data.HeartbeatInterval
		randomSleep := rand.Intn(ws.HeartbeatInterval)
		sendMessage := structs.Message{
			Op: 1,
			D:  &ws.SequenceNum,
		}
		sendMessageJSON, err := json.Marshal(sendMessage)
		if err != nil {
			fmt.Println("Error marshalling message: ", err)
			return
		}
		go func() {
			select {
			case <-time.After(time.Duration(randomSleep) * time.Millisecond):
				err := ws.SendMessage(sendMessageJSON)
				if err != nil {
					fmt.Println("Error sending message:", err)
				}
			case <-ws.ctx.Done():
				fmt.Println("Cancelled delayed heartbeat send")
			}
		}()
	case 1:
		//Heartbeat message, requires immediate heartbeat return
		sendMessage := structs.Message{
			Op: 1,
		}
		sendMessageJSON, err := json.Marshal(sendMessage)
		if err != nil {
			fmt.Println("Error marshalling message: ", err)
			return
		}
		err = ws.SendMessage(sendMessageJSON)
		if err != nil {
			fmt.Println("Error sending message: ", err)
			return
		}
	// case 11:
	// 	sendMessage := structs.Message{
	// 		Op: 1,
	// 		D:  &ws.SequenceNum,
	// 	}
	// 	sendMessageJSON, err := json.Marshal(sendMessage)
	// 	if err != nil {
	// 		fmt.Println("Error marshalling message: ", err)
	// 		return
	// 	}
	// 	go func() {
	// 		time.Sleep(time.Duration(ws.HeartbeatInterval) * time.Millisecond)
	// 		err = ws.SendMessage(sendMessageJSON)
	// 		if err != nil {
	// 			fmt.Println("Error sending message: ", err)
	// 			return
	// 		}
	// 	}()
	case 7:
		fmt.Println("Reconnect request recieved")
		ws.AttemptReconnect()
	case 0:
		//This is where the payload will come from
		ws.SequenceNum = *msg.S
		fmt.Println(ws.SequenceNum)
		switch *msg.T {
		case "READY":
			var data structs.ReadyPayload
			rawData, ok := msg.D.(map[string]interface{})
			if !ok {
				fmt.Println("Error asserting baseMsg.D to map[string]interface{}")
				return
			}
			rawDataBytes, err := json.Marshal(rawData)
			if err != nil {
				fmt.Println("Error marshalling rawData to bytes: ", err)
				return
			}
			err = json.Unmarshal(rawDataBytes, &data)
			if err != nil {
				fmt.Println("Error unmarshalling message: ", err)
				fmt.Println("Message: ", message)
				return
			}
			ws.SequenceNum = *msg.S
			ws.SessionId = data.SessionId
			fmt.Println(string(ws.SessionId))
			ws.ReconnectURL = data.ResumeGatewayURL + "/?v=10&encoding=json"
		case "INTERACTION_CREATE":
			handleInteraction(msg)
		case "RESUMED":
			fmt.Println("Session Resumed")
		case "GUILD_CREATE":
			fmt.Println("Joined guild")
		case "MESSAGE_CREATE":
			handleDiscordMessage(msg)
		default:
			fmt.Println("unknown message received\n" + string(message))
		}
	case 9:
		if msg.D == "true" {
			fmt.Println("Code 9 with d = true recieved, reconnecting")
			ws.AttemptReconnect()
		} else {
			fmt.Println("Session invalid, starting fresh...")
			ws.Close()
			ws.SessionId = ""
			ws.SequenceNum = 0
			ws.ReconnectURL = ""
			ws.Connect(ws.URL)
		}
	}
}

func handleDiscordMessage(msg structs.Message) {
	db := database.GetDB()
	fmt.Println("Message received:")
	var author structs.Author
	rawData, ok := msg.D.(map[string]interface{})
	if !ok {
		fmt.Println("Error asserting message to map[string]interface{}")
		return
	}
	rawDataBytes, err := json.Marshal(rawData)
	if err != nil {
		fmt.Println("Error marshalling raw data")
		return
	}
	err = json.Unmarshal(rawDataBytes, &author)
	if err != nil {
		fmt.Println("Error unmarshalling to Author")
	}
	fmt.Println(author.Id)

	db.DeleteOldSilences()
	db.IsUserSilenced(author.Id)
}

func handleInteraction(message structs.Message) {
	var interaction structs.Interaction
	rawData, ok := message.D.(map[string]interface{})
	if !ok {
		fmt.Println("Error asserting baseMsg.D to map[string]interface{}")
		return
	}
	rawDataBytes, err := json.Marshal(rawData)
	if err != nil {
		fmt.Println("Error marshalling rawData to bytes: ", err)
		return
	}
	err = json.Unmarshal(rawDataBytes, &interaction)
	if err != nil {
		fmt.Println("Error unmarshalling message: ", err)
		fmt.Println("Message: ", message)
		return
	}
	var data structs.InteractionData
	rawData, ok = interaction.Data.(map[string]interface{})
	if !ok {
		fmt.Println("Error asserting interaction to map[string]interface{}")
		return
	}
	rawDataBytes, err = json.Marshal(rawData)
	if err != nil {
		fmt.Println("Error marshalling rawData to bytes: ", err)
		return
	}
	err = json.Unmarshal(rawDataBytes, &data)
	if err != nil {
		fmt.Println("Error unmarshalling message: ", err)
		fmt.Println("Message: ", message)
		return
	}

	switch data.Name {
	case "silence":
		var subCommandOptions []structs.InteractionData
		rawData, ok := data.Options.([]interface{})
		if !ok {
			fmt.Println("Error asserting interaction to map[string]interface{}")
			return
		}
		rawDataBytes, err = json.Marshal(rawData)
		if err != nil {
			fmt.Println("Error marshalling rawData to bytes: ", err)
			return
		}
		err = json.Unmarshal(rawDataBytes, &subCommandOptions)
		if err != nil {
			fmt.Println("Error unmarshalling message: ", err)
			fmt.Println("Message: ", message)
			return
		}
		var options []structs.InteractionDataOptions
		rawData, ok = subCommandOptions[0].Options.([]interface{})
		if !ok {
			fmt.Println("Error asserting interaction to []interface{}")
			return
		}
		rawDataBytes, err = json.Marshal(rawData)
		if err != nil {
			fmt.Println("Error marshalling rawData to bytes: ", err)
			return
		}
		err = json.Unmarshal(rawDataBytes, &options)
		if err != nil {
			fmt.Println("Error unmarshalling message: ", err)
			fmt.Println("Message: ", message)
			return
		}
		mute(options[0].Value.(string), int(options[1].Value.(float64)), interaction.GuildId)
	}
}

func mute(memberId string, minutes int, guildId string) {
	if memberId == "1326247335692341318" {
		fmt.Println("Error, cannot mute bot")
		return
	}
	s := silence.NewSilence(memberId, minutes, guildId)
	s.SilenceUser()
	fmt.Println("User silenced", memberId, minutes)
	//TODO: Get user object of user silenced, start seperate goroutine that server mutes the user, checks to see if they're unmuted
	//then mutes them again if need be. Also delete any messages sent by the muted user
}
