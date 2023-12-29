package broker

type publishChannelContent[TID comparable, TPayload any] struct {
	ID      TID
	Channel chan TPayload
}

type subscribeChannelContent[TID comparable, TPayload any] struct {
	ID      TID
	Channel chan chan TPayload
}

// ChannelBroker passes a channel with ID from producer to the first consumer.
// The subsequent consumers will block until producer is finished so that they
// can resolve the situation e.g. by fetching persisted data from the database.
//
// This kind of broker is useful for streaming chat responses through SSE. The
// Producer in this case is a goroutine spawned by HTTP POST to initiate the streaming.
// The first consumer is the HTTP handler that returns the SSE stream. The subsequent
// consumers are likely caused by connectivity issues. In their case, it's better to
// wait for the producer to finish and return the complete data at the end.
type ChannelBroker[TID comparable, TPayload any] struct {
	stopChannel      chan struct{}
	publishChannel   chan publishChannelContent[TID, TPayload]
	unpublishChannel chan TID
	subscribeChannel chan subscribeChannelContent[TID, TPayload]
}

// NewChannelBroker creates a new ChannelBroker and starts the goroutine that handles it.
// Use Stop() to stop the goroutine.
func NewChannelBroker[TID comparable, TPayload any]() *ChannelBroker[TID, TPayload] {
	broker := ChannelBroker[TID, TPayload]{
		stopChannel:      make(chan struct{}),
		publishChannel:   make(chan publishChannelContent[TID, TPayload]),
		unpublishChannel: make(chan TID),
		subscribeChannel: make(chan subscribeChannelContent[TID, TPayload]),
	}
	return &broker
}

// Start listening for publish, unpublish, and subscribe events. This function blocks until Stop() is called,
// so it should be called in a goroutine. It does not handle panics, so it should be wrapped in a recover.
func (b *ChannelBroker[TID, TPayload]) Start() {
	publishedChannels := map[TID]chan TPayload{}
	subscriberLists := map[TID][]chan chan TPayload{}
	for {
		select {
		case <-b.stopChannel:
			return

		case subscription := <-b.subscribeChannel:
			c := publishedChannels[subscription.ID]
			if c == nil {
				// Signal to the subscriber that the producer is finished (or haven't started yet)
				close(subscription.Channel)
				break
			}
			subscribers := subscriberLists[subscription.ID]
			if subscribers == nil {
				// First subscriber gets the channel from the producer
				subscribers = []chan chan TPayload{subscription.Channel}
				subscription.Channel <- c
			} else {
				// Subsequent subscribers block until the producer is finished
				subscriberLists[subscription.ID] = append(subscribers, subscription.Channel)
			}

		case publication := <-b.publishChannel:
			publishedChannels[publication.ID] = publication.Channel

		case id := <-b.unpublishChannel:
			delete(publishedChannels, id)
			delete(subscriberLists, id)
		}
	}
}

// Stop the goroutine that handles the broker.
func (b *ChannelBroker[TID, TPayload]) Stop() {
	close(b.stopChannel)
}

// Subscribe to the channel with ID. Returns a channel that will receive the channel corresponding to the ID.
// If the channel is not yet published, the returned channel will be closed.
// If there's already a subscriber, the returned channel will block until the producer is finished and then
// close the returned channel.
func (b *ChannelBroker[TID, TPayload]) Subscribe(id TID) chan chan TPayload {
	channel := make(chan chan TPayload, 1)
	b.subscribeChannel <- subscribeChannelContent[TID, TPayload]{
		ID:      id,
		Channel: channel,
	}
	return channel
}

// Publish the channel with ID. The channel will be sent to the first subscriber.
func (b *ChannelBroker[TID, TPayload]) Publish(id TID, channel chan TPayload) {
	b.publishChannel <- publishChannelContent[TID, TPayload]{
		ID:      id,
		Channel: channel,
	}
}

// Unpublish the channel with ID. Note that the channel will be removed from the broker which means
// that subscribers will not be able to receive the channel from the broker. The suggested way to
// get around this is an unbuffered channel that blocks the producer until it gets a consumer. If the
// consumers are unreliable, the producer should have a timeout to not block forever.
func (b *ChannelBroker[TID, TPayload]) Unpublish(id TID) {
	b.unpublishChannel <- id
}
