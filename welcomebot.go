package main

import (
	"os"
    "fmt"
    "strings"
    "encoding/json"
    log "github.com/Sirupsen/logrus"
	"github.com/nlopes/slack"
)

type publicResponse struct {
	Channel	string	`json:"channel"`
	Response	string	`json:"response"`
}

type dmResponse struct {
    Channel string  `json:channel"`
    Response    string  `json:"response"`
}

type Config struct {
    PublicResponses []publicResponse `json:"responses"`
    DmResponses []dmResponse    `json:"dmresponses"`
}

var (
	botId string
)

func main() {

	token := os.Getenv("SLACK_TOKEN")
    config := loadConfig("config.json")
	api := slack.New(token)
	api.SetDebug(false)

	rtm := api.NewRTM()

	go rtm.ManageConnection()

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
                botId = ev.Info.User.ID
				log.Infof("Connection counter:", ev.ConnectionCount)
			case *slack.MessageEvent:
                //only interested in public channels
                cInfo, err := api.GetChannelInfo(ev.Channel)
                if err == nil {
                    if ev.SubType == "channel_join" {
				        log.Infof("channel_join seen on channel: %s", ev.Msg.Channel)
				        respondToJoin(rtm, ev, cInfo.Name, config)
                    }
                    if ev.User != botId && strings.HasPrefix(ev.Text, "<@" + botId + ">") {
                        log.Infof("message seen on public channel: %s", ev.Msg.Channel)
                        respondToMessage(rtm, ev, cInfo.Name, config)
                    }
                }
			case *slack.RTMError:
				log.Errorf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				log.Fatal("Invalid credentials")
				break Loop

			default:
				//Take no action
			}
		}
	}
}


func respondToMessage(rtm *slack.RTM, ev *slack.MessageEvent, name string, config Config) {

	acceptedGreetings := map[string]bool{
		"help": true,
	}

    text := ev.Msg.Text
    prefix := fmt.Sprintf("<@%s> ", botId)
	text = strings.TrimPrefix(text, prefix)
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)

	if acceptedGreetings[text] {
		for _, publicResponse := range config.PublicResponses {
        	if publicResponse.Channel == name {
          	publicMsg := fmt.Sprintf("*Public response for this channel*:\n\n%s", publicResponse.Response)
          	rtm.SendMessage(rtm.NewOutgoingMessage(publicMsg, ev.Msg.Channel))
        	}
    	}

    	for _, dmResponse := range config.DmResponses {
        	if dmResponse.Channel == name {
          	dmMsg := fmt.Sprintf("*DM response for this channel*:\n\n%s", dmResponse.Response)
          	rtm.SendMessage(rtm.NewOutgoingMessage(dmMsg, ev.Msg.Channel))
        	}
    	}
	}
}

func respondToJoin(rtm *slack.RTM, ev *slack.MessageEvent, name string, config Config) {

    
    for _, publicResponse := range config.PublicResponses {
        if publicResponse.Channel == name {
          log.Infof("Sending public reply to channel %s", name)
          rtm.SendMessage(rtm.NewOutgoingMessage(publicResponse.Response, ev.Msg.Channel))
        }
    }

    for _, dmResponse := range config.DmResponses {
        if dmResponse.Channel == name {
          sta, stb, channel, err := rtm.OpenIMChannel(ev.User)
          if err != nil  && sta && stb {
            log.Warnf("Failed to open IM channel to user: %s", err)
          }
          log.Infof("Sending DM to user %s", ev.User)
          rtm.SendMessage(rtm.NewOutgoingMessage(dmResponse.Response, channel))
        }
    }

}

func loadConfig(file string) Config {

    var config Config
    configFile, err := os.Open(file)
    defer configFile.Close()
    if err != nil {
        log.Fatalf("Error opening config file %s", err.Error())
    }
    jsonParser := json.NewDecoder(configFile)
    jsonParser.Decode(&config)
    return config
}
