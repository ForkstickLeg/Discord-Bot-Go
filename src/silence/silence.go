package silence

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	discordapiclient "github.com/ChopstickLeg/Discord-Bot-Practice/src/discord-api-client"
	"github.com/ChopstickLeg/Discord-Bot-Practice/src/structs"
)

type Silence struct {
	userId  string
	time    int
	guildId string
	UserObj structs.GuildMember
}

func NewSilence(id string, timeoutTime int, gId string) *Silence {
	return &Silence{
		userId:  id,
		time:    timeoutTime,
		guildId: gId,
		UserObj: getUserObj(id, gId),
	}
}

func getUserObj(id string, guildId string) structs.GuildMember {
	key := []string{"Content-Type", "Authorization"}
	value := []string{"application/json", "Bot " + os.Getenv("BOT_TOKEN")}

	ac := discordapiclient.NewApiCall("https://discord.com/api/v10/guilds/" + guildId + "/members/" + id)
	ac.AddHeader(key, value)

	output := ac.MakeGetCall()

	var member structs.GuildMember
	err := json.Unmarshal(output, &member)
	if err != nil {
		fmt.Println("Error unmarshalling user data")
	}
	return member
}

func (s *Silence) SilenceUser() {
	timeout := time.Duration(s.time) * time.Minute
	for {
		select {
		case <-time.After(timeout):

			return
		default:
			checkUserStatus(s.userId, s.guildId)
		}
	}
}

func checkUserStatus(id string, guildId string) {
	user := getUserObj(id, guildId)
	if !user.Mute {
		key := []string{"Content-Type", "Authorization"}
		value := []string{"application/json", "Bot " + os.Getenv("BOT_TOKEN")}
		ac := discordapiclient.NewApiCall("https://discord.com/api/v10/guilds/" + guildId + "/members/" + id)
		ac.AddHeader(key, value)
		body := structs.GuildMember{
			Mute: true,
		}
		ac.AddBody(body)
		ac.MakePatchCall()
	}
}
