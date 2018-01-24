// Package auth implements http request authentication
package auth

import (
	"errors"
	"github.com/MG-RAST/AWE/lib/auth/clientgroup"
	"github.com/MG-RAST/AWE/lib/auth/globus"
	"github.com/MG-RAST/AWE/lib/auth/oauth"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/user"
)

// authCache is a
var authCache cache
var authMethods []func(string) (*user.User, error)

func Initialize() {
	authCache = cache{m: make(map[string]cacheValue)}
	authMethods = []func(string) (*user.User, error){}
	if len(conf.AUTH_OAUTH) > 0 {
		authMethods = append(authMethods, oauth.Auth)
	}
	if conf.GLOBUS_TOKEN_URL != "" && conf.GLOBUS_PROFILE_URL != "" {
		authMethods = append(authMethods, globus.Auth)
	}
}

func Authenticate(header string) (u *user.User, err error) {
	if u = authCache.lookup(header); u != nil {
		return u, nil
	} else {
		for _, auth := range authMethods {
			u, err = auth(header)
			if err != nil {
				// log actual error, return consistant invalid auth to user
				last_position := len(header)
				if last_position > 10 {
					last_position = 10
				}
				logger.Error("(auth.Authenticate) err=%s (header=%s)", err.Error(), header[0:last_position]+"...")
				err = nil
			}
			if u != nil {
				authCache.add(header, u)
				return
			}
		}
	}
	return nil, errors.New(e.InvalidAuth)
}

func AuthenticateClientGroup(header string) (cg *core.ClientGroup, err error) {
	if cg, err = clientgroup.Auth(header); err != nil {
		return nil, err
	}
	return cg, nil
}
