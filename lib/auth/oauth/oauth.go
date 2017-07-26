// Package oauth implements generic OAuth authentication
package oauth

import (
	"crypto/tls"
	"encoding/json"
	"errors"
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
func Auth(header string) (*user.User, error) {
	bearer := authHeaderType(header)
	if bearer == "" {
		return nil, errors.New("(oauth) Invalid authentication header, missing bearer token.")
	}
	oauth_url, found_url := conf.AUTH_OAUTH[bearer]
	if bearer == "basic" {
		return nil, errors.New("(oauth) This authentication method does not support username/password authentication. Please use your OAuth token.")
	} else if bearer == "oauth" {
		return authToken(strings.Split(header, " ")[1], conf.OAUTH_DEFAULT)
	} else if found_url {
		return authToken(strings.Split(header, " ")[1], oauth_url)
	} else {
		return nil, errors.New("(oauth) Invalid authentication header, unknown bearer token: " + bearer)
	}
}

// authToken validiates token by fetching user information.
func authToken(token string, url string) (u *user.User, err error) {
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.New("(oauth) HTTP GET: " + err.Error())
	}
	req.Header.Add("Auth", token)
	if resp, err := client.Do(req); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			if body, err := ioutil.ReadAll(resp.Body); err == nil {
				u = &user.User{}
				c := &credentials{}
				if err = json.Unmarshal(body, &c); err != nil {
					return nil, errors.New("(oauth) JSON Unmarshal: " + err.Error())
				} else {
					if c.Uname == "" {
						return nil, errors.New("(oauth) " + e.InvalidAuth)
					}
					u.Username = c.Uname
					if c.Name != "" {
						u.Fullname = c.Name
					} else {
						u.Fullname = c.Fname + " " + c.Lname
					}
					u.Email = c.Email
					if err = u.SetMongoInfo(); err != nil {
						return nil, errors.New("(oauth) MongoDB: " + err.Error())
					}
				}
			}
		} else if resp.StatusCode == http.StatusForbidden {
			return nil, errors.New("(oauth) " + e.InvalidAuth)
		} else {
			return nil, errors.New("(oauth) Authentication failed: Unexpected response status: " + resp.Status)
		}
	} else {
		return nil, errors.New("(oauth) " + err.Error())
	}
	return
}
