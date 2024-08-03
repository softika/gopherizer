package model

import (
	"time"

	"github.com/oklog/ulid/v2"
)

type Base struct {
	Id        ulid.ULID
	CreatedAt time.Time
	UpdatedAt time.Time
}
