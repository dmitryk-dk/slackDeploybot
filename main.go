package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"strconv"

	"github.com/dmitryk-dk/deployBot/config"
	deploybotClient "github.com/dmitryk-dk/deploybotApi/client"
	"github.com/dmitryk-dk/deploybotApi/structs"
	"github.com/dmitryk-dk/deploybotApi/trigger"
	"github.com/dmitryk-dk/slackbot/api"
	slackClient "github.com/dmitryk-dk/slackbot/client"
)

const (
	ServersText      = "server name: %s; \t server id: %d;"
	EnvironmentsText = "environment id: %d; \t environment name: %s; \t used branch: %s"
	UsersText        = "user id: %d; \t user name: %s %s; \t email: %s"
	DeploymentsText  = "Start deployment, success!"
	HelloText        = "Hi, dude! I'm help deploying)!"
	HelpText         = `For creating request use next syntax:
	@botan servers; limit: 20;, where servers - it is the URL query string,
	and after you can put params which deploybot used;
	`
)

var slackToken string
var slackEndpoint string
var deploybotToken string
var deploybotEndpoint string
var botId string

func main() {
	var config config.Config
	slackToken, slackEndpoint, deploybotToken, deploybotEndpoint, botId = config.ReadConfig()

	getUrl := slackEndpoint + slackToken
	resp, err := http.Get(getUrl)

	if err != nil {
		fmt.Errorf("API request failed with code %d", resp.StatusCode)
	}

	if resp.StatusCode != 200 {
		fmt.Errorf("API request failed with code %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	json.Unmarshal(body, &config)

	if err != nil {
		log.Fatal(err)
	}

	clientNew := slackClient.NewClient(slackToken)
	if err := clientNew.Connect(config.SlackWsUrl); err != nil {
		log.Fatal(err)
	}

	clientNew.Loop()

	for {
		select {
		case err := <-clientNew.Errors:
			log.Fatal(err)
		case msg := <-clientNew.Incoming:
			parse(clientNew, msg)
		}
	}
}

func parse(client *slackClient.Client, msg interface{}) {
	switch msg := msg.(type) {

	case api.Hello:
		fmt.Println("Slack says hello!")

	case api.Message:
		channel := msg.Channel
		MessageHandler(channel, &msg, client)

	default:
		fmt.Println("event received", msg)
	}

}

func DeploybotData(action string, params map[string]string) ([]byte, error) {
	client := deploybotClient.Client{}
	act := fmt.Sprintf("/%s", action)

	if action == "deployments" {
		envId, err := strconv.Atoi(params["environment_id"])
		userId, err := strconv.Atoi(params["user_id"])
		dplFromScratch, err := strconv.ParseBool(params["deploy_from_scratch"])
		trgNotiffication, err := strconv.ParseBool(params["trigger_notification"])
		if err != nil {
			log.Fatal("Error converting params to trigger struct: ", err)
		}
		trigger := trigger.Trigger{
			EnvironmentId:       int(envId),
			UserId:              int(userId),
			DeployedVersion:     params["deployed_version"],
			DeployFromScratch:   bool(dplFromScratch),
			TriggerNotification: bool(trgNotiffication),
			Comment:             params["comment"],
		}
		return trigger.TriggerDeployment(act, deploybotEndpoint, deploybotToken)
	}
	return client.GetData(act, deploybotEndpoint, deploybotToken, params)
}

func MessageGenerator(text string, str *structs.ComparedObject, params map[string]string) []string {
	var message = make([]string, 0, 100)
	data, err := DeploybotData(text, params)
	if err != nil {
		log.Fatal(err)
	}

	if err := json.Unmarshal(data, &str); err != nil {
		log.Println("Unmarshal error:", err)
	}

	for _, v := range str.Entries {
		switch val := v.(type) {
		case map[string]interface{}:
			message = append(message, MakeMessages(text, val))
		}
	}
	return message
}

func MakeMessages(action string, val map[string]interface{}) string {
	fmt.Println(action)
	switch action {
	case "servers":
		return fmt.Sprintf(ServersText, val["name"], int(val["id"].(float64)))
	case "environments":
		return fmt.Sprintf(EnvironmentsText, int(val["id"].(float64)), val["name"], val["branch_name"])
	case "users":
		return fmt.Sprintf(UsersText, int(val["id"].(float64)), val["first_name"], val["last_name"], val["email"])
	case "deployments":
		return DeploymentsText
	default:
		return ""
	}
}

func GenerateParams(action []string) map[string]string {
	var param []string
	params := make(map[string]string)
	if len(action) > 0 {
		for i := 1; i < len(action)-1; i++ {
			param = strings.Split(action[i], ":")
			if len(param) > 0 {
				params[strings.TrimSpace(param[0])] = strings.TrimSpace(param[1])
			}
		}
	}
	return params
}

func MessageHandler(channel string, msg *api.Message, client *slackClient.Client) {
	var str string
	var respData structs.ComparedObject

	botId := "<@" + botId + ">"
	isBotUsed := strings.HasPrefix(msg.Text, botId)
	text := strings.TrimSpace(strings.Replace(msg.Text, botId, "", -1))
	action := strings.Split(strings.TrimSpace(text), ";")
	if isBotUsed {
		params := GenerateParams(action)
		switch strings.ToLower(action[0]) {

		case "hello":
			if err := client.SendMessage(channel, HelloText); err != nil {
				log.Println(err)
			}

		case "help":
			if err := client.SendMessage(channel, HelpText); err != nil {
				log.Println(err)
			}

		default:
			msg := MessageGenerator(strings.ToLower(action[0]), &respData, params)
			for _, m := range msg {
				str += m + "\n"
			}
			if err := client.SendMessage(channel, str); err != nil {
				log.Println(err)
			}
		}
	}
}