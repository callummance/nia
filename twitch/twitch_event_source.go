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

type subscription struct {
	StreamOfflineSub string
	StreamOnlineSub  string
}

//EventSource contains a handle to the twitch event listener as well as REST client
type EventSource struct {
	twitchClient      *nazuna.EventsubClient
	liveSubscriptions map[string]subscription
	handler           EventHandler
}

//StartTwitchListener starts listening for events from the Twitch API
func StartTwitchListener(handler EventHandler, initChannelListeners []string) (*EventSource, error) {
	logrus.Tracef("Starting twitch listener with requested Twitch UIDs %v", initChannelListeners)
	opts, err := getOptsFromEnv()
	if err != nil {
		logrus.Errorf("Failed to start twitch listener as %v", err)
		return nil, err
	}
	client, err := nazuna.NewClient(*opts)
	res := EventSource{
		twitchClient:      client,
		liveSubscriptions: make(map[string]subscription, len(initChannelListeners)),
		handler:           handler,
	}

	//Register handlers
	client.RegisterHandler(res.dispatchStreamOnlineEvent)
	client.RegisterHandler(res.dispatchStreamOfflineEvent)

	//	//TODO: check their current state and generate online/offline event to adjust for changes whilst the bot is offline

	//Get current list of subscriptions from API and refresh any that need refreshing
	err = res.refreshSubscriptions()
	if err != nil {
		logrus.Error("Failed to refresh twitch eventsub subscriptions")
		return nil, err
	}
	err = res.SyncSubscriptions(initChannelListeners)
	if err != nil {
		logrus.Error("Failed to resync twitch eventsub subscriptions")
		return nil, err
	}

	return &res, nil
}

//SubscribeToURL takes a twitch name or URL and attempts to subscribe to stream.online and stream.offline events for that broadcaster.
//Returns a twitchStream object if successful.
func (t *EventSource) SubscribeToURL(nameOrURL string) error {
	userData, err := t.twitchClient.GetBroadcaster(nameOrURL)
	if err != nil {
		return err
	}
	broadcasterID := userData.ID
	err = t.SubscribeToStream(broadcasterID)
	if err != nil {
		return err
	}
	return nil
}

//SubscribeToStream attempts to create StreamOnline and StreamOffline subscriptions for the provided broadcaster
//UID. If subscription data already exists in the provided twitchstream struct, this function will do nothing.
func (t *EventSource) SubscribeToStream(twitchUID string) error {
	//Check if we have a subscription registered already
	_, exists := t.liveSubscriptions[twitchUID]
	//Only create a new subscription if one is not already there
	//TODO: check that the subscription is actually still running
	if !exists {
		onlineSub, err := t.twitchClient.CreateSubscription(messages.ConditionStreamOnline{
			BroadcasterUID: twitchUID,
		})
		if err != nil {
			return err
		}
		offlineSub, err := t.twitchClient.CreateSubscription(messages.ConditionStreamOffline{
			BroadcasterUID: twitchUID,
		})
		if err != nil {
			if len(onlineSub.Data) > 0 {
				t.twitchClient.DeleteSubscription(onlineSub.Data[0].ID)
			}
			return err
		}
		//Save subscription details to map
		t.liveSubscriptions[twitchUID] = subscription{
			StreamOnlineSub:  onlineSub.Data[0].ID,
			StreamOfflineSub: offlineSub.Data[0].ID,
		}
	}
	return nil
}

//UnsubscribeFromStream attempts to unsubscribe from stream online and offline events for the provided stream. It will also
//reset the event subscription IDs
func (t *EventSource) UnsubscribeFromStream(twitchUID string) error {
	s, exists := t.liveSubscriptions[twitchUID]
	if !exists {
		return fmt.Errorf("eventsub subscription for twitch stream with ID %v does not exist", twitchUID)
	}
	err := t.twitchClient.DeleteSubscription(s.StreamOfflineSub)
	if err != nil {
		return err
	}
	err = t.twitchClient.DeleteSubscription(s.StreamOnlineSub)
	if err != nil {
		return err
	}
	delete(t.liveSubscriptions, twitchUID)
	return nil
}

//GetBroadcasterDeets looks up a broadcaster by name and attempts to fetch their details
func (t *EventSource) GetBroadcasterDeets(name string) (*restclient.TwitchUser, error) {
	return t.twitchClient.GetBroadcaster(name)
}

//ClearSubscriptions attempts to unsubscribe from all current subscriptions
func (t *EventSource) ClearSubscriptions() error {
	t.liveSubscriptions = make(map[string]subscription)
	err := t.twitchClient.ClearSubscriptions()
	return err
}

//GetStream attempts to retrieve details on an airing stream. Returns nil if an error occurred or if
//nothing was returned by the API (this usually means the stream is not currently live)
func (t *EventSource) GetStream(twitchUID string) (*restclient.TwitchStream, error) {
	res, err := t.twitchClient.GetStreams(restclient.GetStreamsOpts{
		UserID: []string{twitchUID},
	})
	if err != nil {
		return nil, err
	}
	if len(res) < 1 {
		return nil, nil
	}
	return &res[0], nil
}

//ForceStreamUpdate manually checks the status of a given stream and generates a streamonline or streamoffline event.
func (t *EventSource) ForceStreamUpdate(twitchUID string) error {
	stream, err := t.GetStream(twitchUID)
	if err != nil {
		return err
	}
	go func() {
		//Prevent panic from crashing the whole bot
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("Bot handler thread panicked: %v", r)
			}
		}()
		if stream == nil {
			//assume stream is offline
			t.handler.HandleTwitchStreamOffline(&messages.StreamOfflineEvent{
				BroadcasterUID:       twitchUID,
				BroadcasterUserLogin: "unknown",
				BroadcasterUserName:  "unknown",
			})
		} else {
			//stream is online
			t.handler.HandleTwitchStreamOnline(&messages.StreamOnlineEvent{
				BroadcasterUID:       twitchUID,
				BroadcasterUserLogin: stream.UserLogin,
				BroadcasterUserName:  stream.UserName,
				Type:                 stream.Type,
				StartedAt:            stream.StartedAt,
			})
		}
	}()
	return nil
}

//refreshSubscriptions retrieves a new copy of the subscriptions list from the Twitch API, deleting and
//recreating any non-active subscriptions
func (t *EventSource) refreshSubscriptions() error {
	subscriptionsIter := t.twitchClient.Subscriptions(restclient.SubscriptionsParams{})
	for subscription := range subscriptionsIter {
		if subscription.Err != nil {
			return fmt.Errorf("failed to refresh subscriptions as subscription retrieval failed with error %v", subscription.Err)
		}
		subscriptionStatus := subscription.Subscription.Status
		switch subscriptionStatus {
		case "webhook_callback_verification_failed":
			fallthrough
		case "notification_failures_exceeded":
			fallthrough
		case "authorization_revoked":
			fallthrough
		case "user_removed":
			logrus.Infof("Twitch event subscription %v has a non-active status. Recreating...", subscription.Subscription)
			//Subscription is no longer live, so we should cancel it then recreate a new subscription if if is stream.online or stream.offline
			switch subscription.Subscription.Type {
			case "stream.online":
				err := t.twitchClient.DeleteSubscription(subscription.Subscription.ID)
				if err != nil {
					logrus.Errorf("Failed to delete subscription %v whilst refreshing expired subscription due to error %v", subscription.Subscription, err)
				}
				condition := subscription.Subscription.Condition.(map[string]interface{})
				bid := condition["broadcaster_user_id"].(string)
				onlineSub, err := t.twitchClient.CreateSubscription(messages.ConditionStreamOnline{
					BroadcasterUID: bid,
				})
				if err != nil {
					logrus.Errorf("Failed to recreate subscription %v whilst refreshing expired subscription due to error %v", subscription.Subscription, err)
				}
				orig := t.liveSubscriptions[bid]
				orig.StreamOnlineSub = onlineSub.Data[0].ID
				t.liveSubscriptions[bid] = orig
			case "stream.offline":
				err := t.twitchClient.DeleteSubscription(subscription.Subscription.ID)
				if err != nil {
					logrus.Errorf("Failed to delete subscription %v whilst refreshing expired subscription due to error %v", subscription.Subscription, err)
				}
				condition := subscription.Subscription.Condition.(map[string]interface{})
				bid := condition["broadcaster_user_id"].(string)
				offlineSub, err := t.twitchClient.CreateSubscription(messages.ConditionStreamOffline{
					BroadcasterUID: bid,
				})
				if err != nil {
					logrus.Errorf("Failed to recreate subscription %v whilst refreshing expired subscription due to error %v", subscription.Subscription, err)
				}
				orig := t.liveSubscriptions[bid]
				orig.StreamOfflineSub = offlineSub.Data[0].ID
				t.liveSubscriptions[bid] = orig
			}
		case "enabled":
			fallthrough
		case "webhook_callback_verification_pending":
			//If still live, we just need to add to map of subscriptions if necessary
			logrus.Debugf("Adding already-live subscription %v to internal map", subscription.Subscription)
			switch subscription.Subscription.Type {
			case "stream.online":
				condition := subscription.Subscription.Condition.(map[string]interface{})
				bid := condition["broadcaster_user_id"].(string)
				orig := t.liveSubscriptions[bid]
				orig.StreamOnlineSub = subscription.Subscription.ID
				t.liveSubscriptions[bid] = orig
			case "stream.offline":
				condition := subscription.Subscription.Condition.(map[string]interface{})
				bid := condition["broadcaster_user_id"].(string)
				orig := t.liveSubscriptions[bid]
				orig.StreamOfflineSub = subscription.Subscription.ID
				t.liveSubscriptions[bid] = orig
			}
		}
	}
	return nil
}

func (t *EventSource) SyncSubscriptions(desiredSubscriptionUIDs []string) error {
	subs := make(map[string]struct {
		IsRequested bool
		IsLive      bool
	}, len(desiredSubscriptionUIDs))
	for k := range t.liveSubscriptions {
		prev := subs[k]
		prev.IsLive = true
		subs[k] = prev
	}
	for _, s := range desiredSubscriptionUIDs {
		prev := subs[s]
		prev.IsRequested = true
		subs[s] = prev
	}
	for uid, status := range subs {
		switch {
		case status.IsLive && status.IsRequested:
			//Is running and requested, so no need to do anything
			logrus.Debugf("Ignoring already-active subscription to twitch UID %v", uid)
		case status.IsLive && !status.IsRequested:
			//Is running but we don't want it, so delete the subscription
			logrus.Debugf("Unsubscribing from twitch UID %v", uid)
			err := t.UnsubscribeFromStream(uid)
			if err != nil {
				logrus.Errorf("Failed to remove no-longer-required subscription to twitch UID %v due to error %v", uid, err)
			}
		case !status.IsLive && status.IsRequested:
			//Is not currently subscribed but we want notifications so create new subscription
			logrus.Debugf("Adding subscription to twitch UID %v", uid)
			err := t.SubscribeToStream(uid)
			if err != nil {
				logrus.Errorf("Failed to create subscription to twitch UID %v due to error %v", uid, err)
			}
		case !status.IsLive && !status.IsRequested:
			//??????
			logrus.Errorf("Got an impossible case for twitch uid %v?", uid)
		}
	}
	return nil
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
	permissive := false
	logLevel, exists := os.LookupEnv("NIA_LOG_LEVEL")
	if exists && (logLevel == "DEBUG" || logLevel == "TRACE") {
		permissive = true
	}
	opts := nazuna.NazunaOpts{
		WebhookPath:    "/twitchhook",
		ListenOn:       serverPort,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		Scopes:         nil,
		Secret:         "",
		ServerHostname: serverHostname,
		Permissive:     permissive,
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
