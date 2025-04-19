package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/fentezi/export-word/internal/config"
	"github.com/fentezi/export-word/internal/entity"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log/slog"
)

var (
	ErrDocumentNotFound = errors.New("document not found")
)

type Repository struct {
	client *mongo.Client
	logger *slog.Logger
	cfg    config.Mongo
}

func New(ctx context.Context, cfg config.Mongo, logger *slog.Logger) (Repository, error) {
	logger.Info(
		"repository init", slog.String("url", cfg.Address),
		slog.String("database", cfg.Database),
	)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Address))
	if err != nil {
		return Repository{}, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return Repository{}, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return Repository{client: client, cfg: cfg, logger: logger}, nil
}

func (r *Repository) Close(ctx context.Context) error {
	r.logger.Info("closing repository")
	err := r.client.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("failed to close repository: %w", err)
	}

	r.logger.Info("repository closed")
	return nil
}

func (r *Repository) CreateWord(ctx context.Context, msg entity.MongoMessage) error {
	const op = "repository.CreateWord"
	r.logger.Debug("start", slog.String("op", op), slog.Any("message", msg))
	defer r.logger.Debug("end", slog.String("op", op))
	collection := r.client.Database(r.cfg.Database).Collection("words")
	_, err := collection.InsertOne(ctx, msg)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) GetWord(ctx context.Context, word string) (entity.MongoMessage, error) {
	const op = "repository.GetWord"
	r.logger.Debug("start", slog.String("op", op), slog.String("word", word))
	defer r.logger.Debug("end", slog.String("op", op))
	collection := r.client.Database(r.cfg.Database).Collection("words")
	var msg entity.MongoMessage
	err := collection.FindOne(ctx, bson.M{"word": word}).Decode(&msg)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return entity.MongoMessage{}, fmt.Errorf("%s: %w", op, ErrDocumentNotFound)
		}
		return entity.MongoMessage{}, fmt.Errorf("%s: %w", op, err)
	}

	return msg, nil
}

func (r *Repository) GetWords(ctx context.Context) ([]entity.MongoMessage, error) {
	const op = "repository.GetWords"
	r.logger.Debug("start", slog.String("op", op))
	defer r.logger.Debug("end", slog.String("op", op))
	collection := r.client.Database(r.cfg.Database).Collection("words")

	cursor, err := collection.Find(ctx, bson.M{"sent": false})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer cursor.Close(ctx)

	var messages []entity.MongoMessage
	for cursor.Next(ctx) {
		var msg entity.MongoMessage
		if err := cursor.Decode(&msg); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		messages = append(messages, msg)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return messages, nil
}

func (r *Repository) UpdateWord(ctx context.Context, eventID uuid.UUID) error {
	const op = "repository.UpdateWord"
	r.logger.Debug("start", slog.String("op", op), slog.Any("event_id", eventID))
	defer r.logger.Debug("end", slog.String("op", op))
	collection := r.client.Database(r.cfg.Database).Collection("words")

	_, err := collection.UpdateOne(
		ctx, bson.M{"eventId": eventID}, bson.M{"$set": bson.M{"sent": true}},
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) GetWordByEventID(
	ctx context.Context,
	eventID uuid.UUID,
) (entity.MongoMessage, error) {
	const op = "repository.GetWordByEventID"
	r.logger.Debug("start", slog.String("op", op), slog.Any("event_id", eventID))
	defer r.logger.Debug("end", slog.String("op", op))

	collection := r.client.Database(r.cfg.Database).Collection("words")

	var msg entity.MongoMessage
	err := collection.FindOne(ctx, bson.M{"eventId": eventID}).Decode(&msg)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return entity.MongoMessage{}, fmt.Errorf("%s: %w", op, ErrDocumentNotFound)
		}
		return entity.MongoMessage{}, fmt.Errorf("%s: %w", op, err)
	}

	return msg, nil

}
