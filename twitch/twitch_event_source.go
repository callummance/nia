package twitch

import (
	"fmt"
	"os"

	"github.com/callummance/nazuna"
	"github.com/callummance/nazuna/messages"
	"github.com/callummance/nazuna/restclient"
	"github.com/sirupsen/logrus"
)

const (
	twitchClientIDEnvVar     = "NIA_TWITCH_CLIENT_ID"
	twitchClientSecretEnvVar = "NIA_TWITCH_CLIENT_SECRET"
	serverHostnameEnvVar     = "NIA_TWITCH_SERVER_HOSTNAME"
	serverPortEnvVar         = "NIA_TWITCH_SERVER_WH_LISTEN_PORT"
)

//EventHandler is a struct which can handle all the events the discord listener generates.
type EventHandler interface {
	HandleTwitchStreamOnline(*messages.StreamOnlineEvent)
	HandleTwitchStreamOffline(*messages.StreamOfflineEvent)
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
	//TODO: instead of this, keep subscriptions going when bot is closed. This means that we will need to be able to autoremove events
	//from unrecognized twitch UIDs as well as checking and resubscribing to events on initial load
	client.ClearSubscriptions()

	//Register handlers
	client.RegisterHandler(res.dispatchStreamOnlineEvent)
	client.RegisterHandler(res.dispatchStreamOfflineEvent)

	//Subscribe to any events provided
	for _, channelToListenFor := range initChannelListeners {
		logrus.Debugf("Creating suscriptions for stream offline and online events for twitch user %v", channelToListenFor)
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
	err = t.SubscribeToUID(broadcasterID)
	if err != nil {
		return "", err
	}
	return broadcasterID, nil
}

//SubscribeToUID attempts to create StreamOnline and StreamOffline subscriptions for the provided broadcaster
//UID
func (t *EventSource) SubscribeToUID(uid string) error {
	s, err := t.twitchClient.CreateSubscription(messages.ConditionStreamOnline{
		BroadcasterUID: uid,
	})
	if err != nil {
		return err
	}
	_, err = t.twitchClient.CreateSubscription(messages.ConditionStreamOffline{
		BroadcasterUID: uid,
	})
	if err != nil {
		if len(s.Data) > 0 {
			t.twitchClient.DeleteSubscription(s.Data[0].ID)
		}
		return err
	}
	return nil
}

//GetBroadcasterDeets looks up a broadcaster by name and attempts to fetch their details
func (t *EventSource) GetBroadcasterDeets(name string) (*restclient.TwitchUser, error) {
	return t.twitchClient.GetBroadcaster(name)
}

//ClearSubscriptions attempts to unsubscribe from all current subscriptions
func (t *EventSource) ClearSubscriptions() error {
	err := t.twitchClient.ClearSubscriptions()
	return err
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
	serverPort, exists := os.LookupEnv(serverPortEnvVar)
	if !exists {
		logrus.Warnf("`%v` env variable was not set.", serverHostnameEnvVar)
		serverPort = ":8080"
	}
	opts := nazuna.NazunaOpts{
		WebhookPath:    "/twitchhook",
		ListenOn:       serverPort,
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

	//For debugging
	logrus.Debugf("Got stream online alert for stream`%v`\n", ev.BroadcasterUserName)

	//Dispatch to bot handlers
	t.handler.HandleTwitchStreamOnline(ev)
}

func (t *EventSource) dispatchStreamOfflineEvent(s *messages.Subscription, ev *messages.StreamOfflineEvent) {
	//Prevent panic from crashing the whole bot
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Bot handler thread panicked: %v", r)
		}
	}()

	//For debugging
	logrus.Debugf("Got stream offline alert for stream`%v`\n", ev.BroadcasterUserName)

	//Dispatch to bot handlers
	t.handler.HandleTwitchStreamOffline(ev)
}
