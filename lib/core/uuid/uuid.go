package uuid

import (
	gouuid "github.com/MG-RAST/golib/go-uuid/uuid"
)

func NewDeprecated() string {
	return gouuid.New()
}
