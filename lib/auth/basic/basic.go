// Package basic implements basic auth decoding and
// self contained user authentication
package basic

import (
	"encoding/base64"
	"errors"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/user"
	"strings"
)

// DecodeHeader takes the request authorization header and returns
// username and password if it is correctly encoded.
func DecodeHeader(header string) (username string, password string, err error) {
	if strings.ToLower(strings.Split(header, " ")[0]) == "basic" {
		if val, err := base64.URLEncoding.DecodeString(strings.Split(header, " ")[1]); err == nil {
			tmp := strings.Split(string(val), ":")
			if len(tmp) >= 2 {
				return tmp[0], tmp[1], nil
			} else {
				return "", "", errors.New(e.InvalidAuth)
			}
		} else {
			return "", "", errors.New(e.InvalidAuth)
		}

	}
	return "", "", errors.New(e.InvalidAuth)
}

// Auth takes the request authorization header and returns
// user
func Auth(header string) (u *user.User, err error) {
	username, password, err := DecodeHeader(header)
	if err == nil {
		return user.FindByUsernamePassword(username, password)
	}
	return nil, err
}
