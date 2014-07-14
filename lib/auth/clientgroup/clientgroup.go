// Package clientgroup implements clientgroup auth decoding and
// self-contained clientgroup authentication
package clientgroup

import (
	"errors"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"strings"
)

// Auth takes the request authorization header and returns the clientgroup
func Auth(header string) (cg *core.ClientGroup, err error) {
	if strings.ToLower(strings.Split(header, " ")[0]) == "cg_token" {
		token := strings.Split(header, " ")[1]
		cg, err = core.LoadClientGroupByToken(token)
		if err != nil && err.Error() == "not found" {
			return nil, errors.New(e.UnAuth)
		}
		return
	}
	return nil, errors.New(e.InvalidAuth)
}
