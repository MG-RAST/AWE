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

// Initialize _
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

// Authenticate _
func Authenticate(header string) (u *user.User, err error) {
	u = authCache.lookup(header)
	if u != nil {
		return
	}

	for i, auth := range authMethods {
		u, err = auth(header)
		if err != nil {
			// log actual error, return consistant invalid auth to user
			lastPosition := len(header)
			if lastPosition > 10 {
				lastPosition = 10
			}
			logger.Error("(auth.Authenticate) (auth method %d of %d) err=%s (header=%s)", i, len(authMethods), err.Error(), header[0:lastPosition]+"...")
			err = nil
		}
		if u != nil {
			authCache.add(header, u)
			return
		}
	}

	return nil, errors.New(e.InvalidAuth)
}

// AuthenticateClientGroup _
func AuthenticateClientGroup(header string) (cg *core.ClientGroup, err error) {
	if cg, err = clientgroup.Auth(header); err != nil {
		return nil, err
	}
	return cg, nil
}
