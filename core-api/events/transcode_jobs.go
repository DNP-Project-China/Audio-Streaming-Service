package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/segmentio/kafka-go"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			NewTranscodePublisher,
			fx.As(new(TranscodePublisher)),
		),
	),
)

type TranscodePublisher interface {
	PublishCreated(ctx context.Context, trackID string, path string, priority int) error
}

type KafkaTranscodePublisher struct {
	writer *kafka.Writer
}

type transcodeJobCreated struct {
	JobID    string `json:"job_id"`
	TrackID  string `json:"track_id"`
	Path     string `json:"path"`
	Priority int    `json:"priority"`
}

func NewTranscodePublisher(lc fx.Lifecycle, cfg *server.Config) *KafkaTranscodePublisher {
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
	}

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  cfg.KafkaTranscodeTopic,
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireOne,
		Balancer:               &kafka.LeastBytes{},
		WriteTimeout:           cfg.KafkaWriteTimeout,
		ReadTimeout:            cfg.KafkaReadTimeout,
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return writer.Close()
		},
	})

	return &KafkaTranscodePublisher{writer: writer}
}

func (p *KafkaTranscodePublisher) PublishCreated(ctx context.Context, trackID string, path string, priority int) error {
	payload := transcodeJobCreated{
		JobID:    fmt.Sprintf("job-%d", time.Now().UnixNano()),
		TrackID:  trackID,
		Path:     path,
		Priority: priority,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal transcode event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(trackID),
		Value: body,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write transcode event: %w", err)
	}

	return nil
}
