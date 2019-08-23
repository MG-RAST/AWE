package uuid_test

import (
	"fmt"
	. uuid "github.com/MG-RAST/golib/go-uuid/uuid"
	"testing"
)

func TestNew(t *testing.T) {
	newuuid := New()
	fmt.Printf("uuid: %v.\n", newuuid)
}
