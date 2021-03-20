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

const logLevelEnvVar string = "NIA_LOG_LEVEL"
const defaultLogLevel = logrus.InfoLevel

func main() {
	//Load environment
	err := godotenv.Load()
	if err != nil {
		logrus.Warnf("Failed to load .env file due to error %v", err)
	}

	//Set log level from environment
	level, exists := os.LookupEnv(logLevelEnvVar)
	switch {
	case !exists:
		logrus.SetLevel(defaultLogLevel)
	case level == "TRACE":
		logrus.SetLevel(logrus.TraceLevel)
	case level == "DEBUG":
		logrus.SetLevel(logrus.DebugLevel)
	case level == "INFO":
		logrus.SetLevel(logrus.InfoLevel)
	case level == "WARN":
		logrus.SetLevel(logrus.WarnLevel)
	case level == "ERROR":
		logrus.SetLevel(logrus.ErrorLevel)
	}

	//Init bot
	bot, err := bot.Init()
	if err != nil {
		logrus.Fatalf("Failed to start discord bot")
	}
	logrus.Infof("Bot is now running. Press ^+C to exit.")

	//Display bot addition link
	addURL, err := bot.BotAddURL()
	if err != nil {
		logrus.Errorf("Failed to generate bot add URL due to error %v", err)
	} else {
		fmt.Printf("Go to `%v` to add bot to your server", addURL)
	}

	//Wait for system interrupt
	closeChan := make(chan os.Signal, 1)
	signal.Notify(closeChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-closeChan

	bot.Close()
	fmt.Println("Goodbye!")
}
