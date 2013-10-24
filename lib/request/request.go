package request

import (
	"errors"
	"github.com/MG-RAST/AWE/lib/auth"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/goweb"
	"net/http"
)

func Authenticate(req *http.Request) (u *user.User, err error) {
	if _, ok := req.Header["Authorization"]; !ok {
		err = errors.New(e.NoAuth)
		return
	}
	header := req.Header.Get("Authorization")
	u, err = auth.Authenticate(header)
	return
}

func RetrieveToken(req *http.Request) (token string, err error) {
	if _, ok := req.Header["Datatoken"]; !ok {
		err = errors.New("no token received")
		return
	}
	token = req.Header.Get("Datatoken")
	return
}

func AuthError(err error, cx *goweb.Context) {
	if err.Error() == e.InvalidAuth {
		cx.RespondWithErrorMessage("Invalid authorization header or content", http.StatusBadRequest)
		return
	}
	logger.Error("Error at Auth: " + err.Error())
	cx.RespondWithError(http.StatusInternalServerError)
	return
}
