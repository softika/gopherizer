package internal

import (
	"time"
)

type Base struct {
	Id        string    `db:"id,pk"` // uuid
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
