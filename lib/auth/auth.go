// Package auth implements http request authentication
package auth

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/auth/basic"
	"github.com/MG-RAST/AWE/lib/auth/globus"
	"github.com/MG-RAST/AWE/lib/auth/mgrast"
	"github.com/MG-RAST/AWE/lib/conf"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/user"
)

// authCache is a
var authCache cache
var authMethods []func(string) (*user.User, error)

func Initialize() {
	fmt.Printf("auth.Initialize=\n")
	authCache = cache{m: make(map[string]cacheValue)}
	authMethods = []func(string) (*user.User, error){}
	if conf.BASIC_AUTH {
		authMethods = append(authMethods, basic.Auth)
	}
	fmt.Printf("globus auth=%t\n", conf.GLOBUS_OAUTH)
	if conf.GLOBUS_OAUTH {
		fmt.Printf("globus auth method\n")
		authMethods = append(authMethods, globus.Auth)
	}
	if conf.MGRAST_OAUTH {
		authMethods = append(authMethods, mgrast.Auth)
	}
	fmt.Printf("authMethod=%v\n", authMethods)
}

func Authenticate(header string) (u *user.User, err error) {
	if u = authCache.lookup(header); u != nil {
		return u, nil
	} else {
		for _, auth := range authMethods {
			if u, _ := auth(header); u != nil {
				authCache.add(header, u)
				return u, nil
			}
		}
	}
	return nil, errors.New(e.InvalidAuth)
}
