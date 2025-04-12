package natsio

import (
	"context"
	"encoding/json"
	"github.com/nats-io/nats.go"
	"log/slog"
	"time"
)

const (
	exampleSubjectPattern = "example.>"
	exampleQueueGroup     = "example-worker-group"
)

type ExampleMessage struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type ExampleSubscriber struct {
	nc     *nats.Conn
	logger *slog.Logger
	sub    *nats.Subscription
}

func NewExampleSubscriber(nc *nats.Conn, logger *slog.Logger) *ExampleSubscriber {
	return &ExampleSubscriber{
		nc:     nc,
		logger: logger,
	}
}

func (s *ExampleSubscriber) Subscribe(ctx context.Context) error {
	s.logger.Info("subscribing to example messages",
		"subject", exampleSubjectPattern,
		"queue_group", exampleQueueGroup)

	sub, err := s.nc.QueueSubscribe(exampleSubjectPattern, exampleQueueGroup, func(msg *nats.Msg) {
		s.handleMessage(ctx, msg)
	})

	if err != nil {
		return err
	}

	s.sub = sub
	return nil
}

func (s *ExampleSubscriber) handleMessage(ctx context.Context, msg *nats.Msg) {
	s.logger.Info("received message", "subject", msg.Subject, "reply", msg.Reply)

	var exampleMsg ExampleMessage
	if err := json.Unmarshal(msg.Data, &exampleMsg); err != nil {
		s.logger.Error("failed to unmarshal message", "error", err, "data", string(msg.Data))
		return
	}

	s.logger.Info("processing message",
		"id", exampleMsg.ID,
		"text", exampleMsg.Text,
		"timestamp", exampleMsg.Timestamp)

	// Example of how to reply if needed
	if msg.Reply != "" {
		response := []byte(`{"status":"processed"}`)
		if err := msg.Respond(response); err != nil {
			s.logger.Error("failed to respond to message", "error", err)
		}
	}
}

func (s *ExampleSubscriber) Unsubscribe() error {
	if s.sub != nil {
		s.logger.Info("unsubscribing from example messages")
		return s.sub.Unsubscribe()
	}
	return nil
}
