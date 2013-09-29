package globus_test

import (
	"fmt"
	. "github.com/MG-RAST/AWE/lib/auth/globus"
	"testing"
)

var (
	u = "papa"
	p = "papa"
)

func TestFetchToken(t *testing.T) {
	token, err := FetchToken(u, p)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		fmt.Printf("%#v\n", token)
	}
}

func TestFetchProfile(t *testing.T) {
	token, err := FetchToken(u, p)
	if err != nil {
		t.Fatal(err.Error())
	}
	user, err := FetchProfile(token.AccessToken)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		fmt.Printf("%#v\n", user)
	}
}
