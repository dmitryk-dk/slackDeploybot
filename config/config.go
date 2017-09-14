package config

import (
	"encoding/json"
	"flag"
	"io/ioutil"
)

type Config struct {
	SlackEndpoint     string `json:"slack_endpoint"`
	SlackToken        string `json:"slack_token"`
	DeploybotEndpoint string `json:"deploybot_endpoint"`
	DeploybotToken    string `json:"deploybot_token"`
	BotId             string `json:"bot_id"`
	SlackWsUrl        string `json:"url"`
}

func (c *Config) ReadConfig() (string, string, string, string, string) {
	flagName := c.ReadFlags()
	file, err := ioutil.ReadFile(*flagName)
	err = json.Unmarshal(file, &c)
	if err != nil {
		return "", "", "", "", ""
	}
	return c.SlackToken, c.SlackEndpoint, c.DeploybotToken, c.DeploybotEndpoint, c.BotId
}

func (c *Config) ReadFlags() (configFile *string) {
	configFile = flag.String("config", "config.json", "Path to config file")
	flag.Parse()
	return
}
