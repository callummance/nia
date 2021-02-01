package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/callummance/nia/bot"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		logrus.Warnf("Failed to load .env file due to error %v", err)
	}
	bot, err := bot.Init()
	if err != nil {
		logrus.Fatalf("Failed to start discord bot")
	}
	logrus.Infof("Bot is now running. Press ^+C to exit.")
	addURL, err := bot.BotAddURL()
	if err != nil {
		logrus.Errorf("Failed to generate bot add URL due to error %v", err)
	} else {
		logrus.Infof("Go to `%v` to add bot to your server", addURL)
	}
	closeChan := make(chan os.Signal, 1)
	signal.Notify(closeChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-closeChan

	bot.Close()
	fmt.Println("Goodbye!")
}
