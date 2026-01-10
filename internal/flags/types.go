package flags

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("flag not found")

type Flag struct {
	Key       string    `json:"key"`
	Value     bool      `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
