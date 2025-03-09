package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fentezi/export-word/config"
	"github.com/fentezi/export-word/internal/entity"
	"github.com/fentezi/export-word/internal/gmail"
	"github.com/fentezi/export-word/internal/kafka"
	"github.com/fentezi/export-word/internal/repository"
	"log/slog"
	"os"
	"sync"
	"time"
)

const (
	wordsFileName = "words.txt"
)

type Service struct {
	log       *slog.Logger
	cfg       *config.Config
	broker    *kafka.Consumer
	gm        *gmail.Gmail
	repo      *repository.Repository
	wordCount int
}

// New creates a new Service instance with the provided dependencies.
func New(
	log *slog.Logger, cfg *config.Config, broker *kafka.Consumer, gm *gmail.Gmail,
	repo *repository.Repository,
) *Service {
	return &Service{
		log:    log,
		cfg:    cfg,
		broker: broker,
		gm:     gm,
		repo:   repo,
	}
}

// Run starts the service, consuming messages from Kafka and periodically writing words to a
// file and sending it via email.
func (s *Service) Run(ctx context.Context) error {
	topic := s.cfg.Kafka.Topic

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		s.consumeMessages(ctx, topic)
	}()

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				s.log.Info("stop writeWordsToFileAndSend goroutine")
				return
			case <-time.After(120 * time.Second):
				s.writeWordsToFileAndSend(ctx)
			}
		}
	}()
	s.log.Info("wait goroutine")
	wg.Wait()
	s.log.Info("finish goroutine")
	return nil
}

// consumeMessages reads from Kafka, processes messages, and writes to a file
func (s *Service) consumeMessages(ctx context.Context, topic string) {
	ch, err := s.broker.Consume(ctx, topic)
	if err != nil {
		s.log.Error("failed to consume messages", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			s.log.Info("stop consume messages")
			return
		case msg := <-ch:
			s.log.Debug("get message", slog.String("message", string(msg)))
			if err := s.processMessage(ctx, msg); err != nil {
				s.log.Error("failed to process message", "error", err)
			}
		}
	}
}

// processMessage processes a single message from Kafka,
// saving it to the database if it's a new event.
func (s *Service) processMessage(ctx context.Context, msg []byte) error {
	const op = "service.processMessage"

	m, err := toKafkaMessage(msg)
	if err != nil {
		s.log.Error(
			"failed to decode message", slog.String("error", err.Error()),
			slog.String("message", string(msg)),
		)
		return fmt.Errorf("%s: %w", op, err)
	}
	s.log.Debug("decode message", slog.Any("message", m))

	_, err = s.repo.GetWordByEventID(ctx, m.EventID)
	if err != nil {
		if errors.Is(err, repository.ErrDocumentNotFound) {
			s.log.Info("message not found, creating", slog.Any("message", m))
			if err := s.repo.CreateWord(ctx, toMongoMessage(m)); err != nil {
				s.log.Error(
					"failed to save message to database", slog.String("error", err.Error()),
					slog.Any("message", m),
				)
				return fmt.Errorf("failed to save message to database: %w", err)
			}
			s.log.Info("message created", slog.Any("message", m))
			s.log.Info("processed message", "word", m.Word, "translation", m.Translation)
			return nil
		}
		s.log.Error(
			"failed to get word by event id", slog.String("error", err.Error()),
			slog.Any("message", m),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	s.log.Info("processed message", "word", m.Word, "translation", m.Translation)
	return nil
}

// writeWordsToFileAndSend writes the words from the database to a file and sends the file via
// email.
func (s *Service) writeWordsToFileAndSend(ctx context.Context) {
	file, err := initFile()
	if err != nil {
		s.log.Error("failed to initialize file", "error", err)
		return
	}
	defer file.Close()
	s.log.Debug("file initialize")

	if err := s.writeWordsToFile(ctx, file); err != nil {
		s.log.Error("failed to write words to file", "error", err)
		return
	}
	s.log.Info("words write to file")

	if s.wordCount == 0 {
		s.log.Info("no words to send")
		return
	}

	msg := gmail.Message{
		From:    s.cfg.Gmail.Email,
		To:      []string{s.cfg.Gmail.Email},
		Subject: "Dictionary",
		Body:    "List words",
		File:    wordsFileName,
	}
	if err := s.gm.SendMessage(msg); err != nil {
		s.log.Error("failed to send message", "error", err)
		return
	}
	s.log.Info("message send")
}

// writeWordsToFile retrieves words from the database and writes them to the specified file.
func (s *Service) writeWordsToFile(ctx context.Context, file *os.File) error {
	const op = "service.writeWordsToFile"

	words, err := s.repo.GetWords(ctx)
	if err != nil {
		s.log.Error("failed to get words from database", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}
	s.log.Debug("get words count", slog.Int("count", len(words)))

	s.wordCount = 0

	for _, word := range words {
		line := fmt.Sprintf("%s;%s\n", word.Word, word.Translation)
		if _, err := file.WriteString(line); err != nil {
			s.log.Error(
				"failed to write to file", slog.String("error", err.Error()),
				slog.String("line", line),
			)
			return fmt.Errorf("%s: %w", op, err)
		}
		if err := s.repo.UpdateWord(ctx, word.EventID); err != nil {
			s.log.Error(
				"failed to update word", slog.String("error", err.Error()),
				slog.Any("word", word),
			)
			return fmt.Errorf("%s: %w", op, err)
		}
		s.wordCount++
	}

	s.log.Info("words written to file", "count", len(words))

	return nil
}

// toKafkaMessage decodes JSON bytes into KafkaMessage struct
func toKafkaMessage(msg []byte) (entity.KafkaMessage, error) {
	const op = "service.toKafkaMessage"
	var m entity.KafkaMessage
	err := json.Unmarshal(msg, &m)
	if err != nil {
		return entity.KafkaMessage{}, fmt.Errorf("%s: %w", op, err)
	}
	return m, nil
}

// toMongoMessage converts KafkaMessage to MongoMessage format
func toMongoMessage(msg entity.KafkaMessage) entity.MongoMessage {
	return entity.MongoMessage{
		EventID:     msg.EventID,
		Word:        msg.Word,
		Translation: msg.Translation,
		Sent:        false,
	}
}

// initFile creates a new file with the name wordsFileName.
func initFile() (*os.File, error) {
	const op = "service.initFile"
	file, err := os.OpenFile(wordsFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("%s: open file: %w", op, err)
	}

	return file, nil
}
