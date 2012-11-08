package uuid

import (
	gouuid "code.google.com/p/go-uuid/uuid"
)

func New() string {
	return gouuid.New()
}
