package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/segmentio/kafka-go"
)

func TestKafkaTranscodePublisher_PublishCreated_WritesTopicMessage(t *testing.T) {
	cfg, err := server.NewConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	brokers := strings.Split(cfg.KafkaBrokers, ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
	}
	if len(brokers) == 0 || brokers[0] == "" {
		t.Skip("kafka brokers are not configured")
	}

	if !kafkaReachable(brokers[0]) {
		t.Skipf("kafka is not reachable at %s", brokers[0])
	}

	topic := fmt.Sprintf("transcode-jobs-test-%d", time.Now().UnixNano())
	if err := ensureTopic(brokers[0], topic); err != nil {
		t.Fatalf("create kafka topic: %v", err)
	}

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		AllowAutoTopicCreation: false,
		RequiredAcks:           kafka.RequireOne,
		Balancer:               &kafka.LeastBytes{},
		WriteTimeout:           10 * time.Second,
		ReadTimeout:            10 * time.Second,
	}
	defer writer.Close()

	pub := &KafkaTranscodePublisher{writer: writer}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	trackID := fmt.Sprintf("track-%d", time.Now().UnixNano())
	path := "raw/track-test/original.mp3"
	if err := pub.PublishCreated(ctx, trackID, path, 1); err != nil {
		t.Fatalf("publish transcode job: %v", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("read kafka message: %v", err)
	}

	if string(msg.Key) != trackID {
		t.Fatalf("unexpected kafka key: got=%q want=%q", string(msg.Key), trackID)
	}

	var payload transcodeJobCreated
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		t.Fatalf("decode kafka payload: %v", err)
	}

	if payload.TrackID != trackID {
		t.Fatalf("payload track_id mismatch: got=%q want=%q", payload.TrackID, trackID)
	}
	if payload.Path != path {
		t.Fatalf("payload path mismatch: got=%q want=%q", payload.Path, path)
	}
	if payload.Priority != 1 {
		t.Fatalf("payload priority mismatch: got=%d want=1", payload.Priority)
	}
	if payload.JobID == "" {
		t.Fatalf("payload job_id is empty")
	}
}

func kafkaReachable(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func ensureTopic(addr string, topic string) error {
	host, portRaw, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	conn, err := kafka.Dial("tcp", net.JoinHostPort(host, portRaw))
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	topicCfg := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}

	if err := controllerConn.CreateTopics(topicCfg); err != nil {
		return err
	}

	return nil
}
