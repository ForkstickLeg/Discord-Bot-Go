package silence

import (
	"encoding/json"
	"fmt"
	"os"

	discordapiclient "github.com/ChopstickLeg/Discord-Bot-Practice/src/discord-api-client"
	"github.com/ChopstickLeg/Discord-Bot-Practice/src/structs"
)

type Silence struct {
	userId  string
	time    int
	userObj structs.User
}

func NewSilence(id string, time int) *Silence {
	return &Silence{
		userId:  id,
		time:    time,
		userObj: getUserObj(id),
	}
}

func getUserObj(id string) structs.User {
	key := []string{"Content-Type", "Authorization"}
	value := []string{"application/json", "Bot " + os.Getenv("BOT_TOKEN")}

	ac := discordapiclient.NewApiCall("https://discord.com/api/v10/users/" + id)
	ac.AddHeader(key, value)

	output := ac.MakeGetCall()

	var user structs.User
	err := json.Unmarshal(output, &user)
	if err != nil {
		fmt.Println("Error unmarshalling user data")
	}
	return user
}

func (s *Silence) SilenceUser() {

}
