package uuid

import (
	gouuid "github.com/MG-RAST/golib/go-uuid/uuid"
)

func New() string {
	return gouuid.New()
}
