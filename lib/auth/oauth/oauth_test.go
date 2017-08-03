package oauth_test

import (
	"fmt"
	. "github.com/MG-RAST/AWE/lib/auth/oauth"
	"github.com/MG-RAST/AWE/lib/conf"
	"testing"
)

var (
	valid   = "S9RH9fP7nh4bPEdUwf2fm4CML"
	invalid = "this_is_not_valid"
)

func init() {
	conf.Initialize()
}

func TestAuthToken(t *testing.T) {
	user, err := AuthToken(valid)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		fmt.Printf("%#v\n", user)
	}
	user, err = AuthToken(invalid)
	if err == nil {
		t.Fatal("Invalid token not returning error.")
	} else {
		fmt.Printf("Invalid token failing: success")
	}
}
