// Package oauth implements generic OAuth authentication
package oauth

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/user"
	"io/ioutil"
	"net/http"
	"strings"
)

type resErr struct {
	error string `json:"error"`
}

type credentials struct {
	Uname string `json:"login"`
	Name  string `json:"name"`
	Fname string `json:"firstname"`
	Lname string `json:"lastname"`
	Email string `json:"email"`
}

func authHeaderType(header string) string {
	tmp := strings.Split(header, " ")
	if len(tmp) > 1 {
		return strings.ToLower(tmp[0])
	}
	return ""
}

// Auth takes the request authorization header and returns user
// bearer token "oauth" returns default url (first item in auth_oauth_url conf value)
func Auth(header string) (u *user.User, err error) {
	bearer := authHeaderType(header)
	if bearer == "" {
		return nil, errors.New("(Auth) Invalid authentication header, missing bearer token.")
	}
	oauth_url, found_url := conf.AUTH_OAUTH[bearer]
	if bearer == "basic" {
		err = errors.New("(Auth) This authentication method does not support username/password authentication. Please use your OAuth token.")
		return
	} else if bearer == "oauth" {
		u, err = authToken(strings.Split(header, " ")[1], conf.OAUTH_DEFAULT)
		if err != nil {
			err = fmt.Errorf("(Auth) bearer=oauth error: %s", err.Error())
		}
		return
	} else if found_url {
		u, err = authToken(strings.Split(header, " ")[1], oauth_url)
		if err != nil {
			err = fmt.Errorf("(Auth) found_url=true error: %s", err.Error())
		}
		return
	} else {
		return nil, errors.New("(Auth) Invalid authentication header, unknown bearer token: " + bearer)
	}
}

// authToken validiates token by fetching user information.
func authToken(token string, url string) (u *user.User, err error) {
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.New("(authToken) HTTP GET: " + err.Error())
	}
	req.Header.Add("Auth", token)
	resp, err := client.Do(req)
	if err != nil {
		err = errors.New("(authToken) " + err.Error())
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		err = errors.New("(authToken) " + e.InvalidAuth)
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("(authToken) Authentication failed: Unexpected response status: %s", resp.Status)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("(authToken) ioutil.ReadAll(resp.Body) failed: %s", err.Error())
		return
	}

	u = &user.User{}
	c := &credentials{}
	err = json.Unmarshal(body, &c)
	if err != nil {
		u = nil
		err = errors.New("(authToken) JSON Unmarshal: " + err.Error())
		return
	}

	if c.Uname == "" {
		u = nil
		err = errors.New("(authToken) c.Uname is empty, " + e.InvalidAuth)
		return
	}
	u.Username = c.Uname
	if c.Name != "" {
		u.Fullname = c.Name
	} else {
		u.Fullname = c.Fname + " " + c.Lname
	}
	u.Email = c.Email
	err = u.SetMongoInfo()
	if err != nil {
		u = nil
		err = errors.New("(authToken) MongoDB: " + err.Error())
		return
	}

	return
}
