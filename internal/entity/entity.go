package entity

import (
	"github.com/google/uuid"
)

type KafkaMessage struct {
	EventID     uuid.UUID `json:"event_id"`
	Word        string    `json:"word"`
	Translation string    `json:"translation"`
}

type MongoMessage struct {
	EventID     uuid.UUID `bson:"eventId"`
	Word        string    `bson:"word"`
	Translation string    `bson:"translation"`
	Sent        bool      `bson:"sent"`
}
