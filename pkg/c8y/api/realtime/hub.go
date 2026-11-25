package realtime

import "sync"

// Hub maintains the set of active subscriptions and broadcasts messages to the clients.
type Hub struct {
	// Registered clients.
	clients map[*subscription]bool

	// Inbound messages from the clients.
	broadcast chan *Message

	// Register requests from the clients.
	register chan *subscription

	// Unregister requests from clients by channel name.
	unregister chan string

	// Return a channel of channels
	getChannels chan chan string

	// Closes the out channel of every subscription and removes them.
	// Used when the client fails permanently so that subscribers are released.
	closeAll chan struct{}

	channels sync.Map
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan *Message),
		register:   make(chan *subscription),
		unregister: make(chan string),
		clients:    make(map[*subscription]bool),

		getChannels: make(chan chan string),
		closeAll:    make(chan struct{}),
	}
}

// GetActiveChannels returns the list of active channels which are currently subscribed to
func (h *Hub) GetActiveChannels() []string {
	channels := []string{}
	h.channels.Range(func(key, value any) bool {
		channels = append(channels, key.(string))
		return true
	})
	return channels
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.channels.Store(client.glob.String(), true)
		case channel := <-h.unregister:
			// Delete clients with the same channel pattern
			h.channels.Delete(channel)
			for client := range h.clients {
				if client.glob.String() == channel {
					delete(h.clients, client)
				}
			}
		case <-h.closeAll:
			// A channel can be registered more than once (subscriptions are
			// registered again after a reconnect): close it once.
			closed := make(map[chan<- *Message]struct{}, len(h.clients))
			for client := range h.clients {
				if _, done := closed[client.out]; !done {
					close(client.out)
					closed[client.out] = struct{}{}
				}
				delete(h.clients, client)
			}
			h.channels.Range(func(key, _ any) bool {
				h.channels.Delete(key)
				return true
			})
		case message := <-h.broadcast:
			for client := range h.clients {
				if ((client.isWildcard && client.glob.MatchString(message.Channel)) || client.glob.String() == message.Channel) && !client.disabled {
					client.out <- message
				}
			}
		}
	}
}
