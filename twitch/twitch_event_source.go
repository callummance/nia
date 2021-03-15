package twitch

import (
	"fmt"
	"os"

	"github.com/callummance/nazuna"
	"github.com/callummance/nazuna/messages"
	"github.com/sirupsen/logrus"
)

const (
	twitchClientIDEnvVar     = "NIA_TWITCH_CLIENT_ID"
	twitchClientSecretEnvVar = "NIA_TWITCH_CLIENT_SECRET"
	serverHostnameEnvVar     = "NIA_SERVER_HOSTNAME"
)

//EventHandler is a struct which can handle all the events the discord listener generates.
type EventHandler interface {
	HandleStreamOnline(*messages.StreamOnlineEvent)
	HandleStreamOffline(*messages.StreamOfflineEvent)
}

type EventSource struct {
	twitchClient *nazuna.EventsubClient
	handler      EventHandler
}

func StartTwitchListener(handler EventHandler, initChannelListeners []string) (*EventSource, error) {
	opts, err := getOptsFromEnv()
	if err != nil {
		logrus.Errorf("Failed to start twitch listener as %v", err)
		return nil, err
	}
	client, err := nazuna.NewClient(*opts)
	res := EventSource{
		twitchClient: client,
		handler:      handler,
	}

	//Clear any old subscriptions
	client.ClearSubscriptions()

	//Register handlers
	client.RegisterHandler(res.dispatchStreamOnlineEvent)
	client.RegisterHandler(res.dispatchStreamOfflineEvent)

	//Subscribe to any events provided
	for _, channelToListenFor := range initChannelListeners {
		//Register stream.online subscription
		client.CreateSubscription(messages.ConditionStreamOnline{
			BroadcasterUID: channelToListenFor,
		})
		//Register stream.offline subscription
		client.CreateSubscription(messages.ConditionStreamOffline{
			BroadcasterUID: channelToListenFor,
		})

		//TODO: check their current state and generate online/offline event to adjust for changes whilst the bot is offline
	}
	return &res, nil
}

//SubscribeToURL takes a twitch name or URL and attempts to subscribe to stream.online and stream.offline events for that broadcaster.
//Returns the broadcaster ID if successful.
func (t *EventSource) SubscribeToURL(nameOrURL string) (string, error) {
	userData, err := t.twitchClient.GetBroadcaster(nameOrURL)
	if err != nil {
		return "", err
	}
	broadcasterID := userData.ID
	s, err := t.twitchClient.CreateSubscription(messages.ConditionStreamOnline{
		BroadcasterUID: broadcasterID,
	})
	if err != nil {
		return "", err
	}
	_, err = t.twitchClient.CreateSubscription(messages.ConditionStreamOffline{
		BroadcasterUID: broadcasterID,
	})
	if err != nil {
		if len(s.Data) > 0 {
			t.twitchClient.DeleteSubscription(s.Data[0].ID)
		}
		return "", err
	}

	return broadcasterID, nil
}

func getOptsFromEnv() (*nazuna.NazunaOpts, error) {
	clientID, exists := os.LookupEnv(twitchClientIDEnvVar)
	if !exists {
		logrus.Warnf("`%v` env variable was not set.", twitchClientIDEnvVar)
		return nil, fmt.Errorf("`%v` env variable was not set", twitchClientIDEnvVar)
	}
	clientSecret, exists := os.LookupEnv(twitchClientSecretEnvVar)
	if !exists {
		logrus.Warnf("`%v` env variable was not set.", twitchClientSecretEnvVar)
		return nil, fmt.Errorf("`%v` env variable was not set", twitchClientSecretEnvVar)
	}
	serverHostname, exists := os.LookupEnv(serverHostnameEnvVar)
	if !exists {
		logrus.Warnf("`%v` env variable was not set.", serverHostnameEnvVar)
		return nil, fmt.Errorf("`%v` env variable was not set", serverHostnameEnvVar)
	}
	opts := nazuna.NazunaOpts{
		WebhookPath:    "/twitchhook",
		ListenOn:       ":8080",
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		Scopes:         nil,
		Secret:         "",
		ServerHostname: serverHostname,
	}
	return &opts, nil
}

func (t *EventSource) dispatchStreamOnlineEvent(s *messages.Subscription, ev *messages.StreamOnlineEvent) {
	//Prevent panic from crashing the whole bot
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Bot handler thread panicked: %v", r)
		}
	}()

	//Dispatch to bot handlers
	t.handler.HandleStreamOnline(ev)

	//For debugging
	logrus.Debugf("Got stream online alert for stream`%v`\n", ev.BroadcasterUserName)
}

func (t *EventSource) dispatchStreamOfflineEvent(s *messages.Subscription, ev *messages.StreamOfflineEvent) {
	//Prevent panic from crashing the whole bot
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Bot handler thread panicked: %v", r)
		}
	}()

	//Dispatch to bot handlers
	t.handler.HandleStreamOffline(ev)

	//For debugging
	logrus.Debugf("Got stream offline alert for stream`%v`\n", ev.BroadcasterUserName)
}
