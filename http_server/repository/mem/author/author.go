package author

import (
	"time"

	"github.com/google/uuid"
)

type Author struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

func NewAuthor(name string) *Author {
	return &Author{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: time.Now(),
	}
}
