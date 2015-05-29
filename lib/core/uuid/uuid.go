package uuid

import (
	gouuid "github.com/MG-RAST/AWE/vendor/code.google.com/p/go-uuid/uuid"
)

func New() string {
	return gouuid.New()
}
