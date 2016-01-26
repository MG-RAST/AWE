package uuid

import (
	gouuid "github.com/pborman/uuid"
)

func New() string {
	return gouuid.New()
}
