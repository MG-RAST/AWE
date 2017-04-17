// Package globus implements Globus Online Nexus authentication
//(code is modified from github.com/MG-RAST/Shock/shock-server/auth/globus)
package globus

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/MG-RAST/AWE/lib/auth/basic"
	"github.com/MG-RAST/AWE/lib/conf"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/user"
)

// Token response struct
type token struct {
	AccessToken     string      `json:"access_token"`
	AccessTokenHash string      `json:"access_token_hash"`
	ClientId        string      `json:"client_id"`
	ExpiresIn       int         `json:"expires_in"`
	Expiry          int         `json:"expiry"`
	IssuedOn        int         `json:"issued_on"`
	Lifetime        int         `json:"lifetime"`
	Scopes          interface{} `json:"scopes"`
	TokenId         string      `json:"token_id"`
	TokeType        string      `json:"token_type"`
	UserName        string      `json:"user_name"`
}

func authHeaderType(header string) string {
	tmp := strings.Split(header, " ")
	if len(tmp) > 1 {
		return strings.ToLower(tmp[0])
	}
	return ""
}

// Auth takes the request authorization header and returns
// user
func Auth(header string) (usr *user.User, err error) {
	switch authHeaderType(header) {
	case "globus-goauthtoken", "oauth":
		return fetchProfile(strings.Split(header, " ")[1])
	case "basic":
		if username, password, err := basic.DecodeHeader(header); err == nil {
			if t, err := fetchToken(username, password); err == nil {
				return fetchProfile(t.AccessToken)
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	default:
		return nil, errors.New("Invalid authentication header.")
	}
	return nil, errors.New("Invalid authentication header.")
}

// fetchToken takes username and password and then retrieves user token
func fetchToken(u string, p string) (t *token, err error) {
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	req, err := http.NewRequest("GET", conf.GLOBUS_TOKEN_URL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(u, p)
	if resp, err := client.Do(req); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			if body, err := ioutil.ReadAll(resp.Body); err == nil {
				if err = json.Unmarshal(body, &t); err != nil {
					return nil, err
				}
			}
		} else {
			return nil, errors.New("Authentication failed: Unexpected response status: " + resp.Status)
		}
	} else {
		return nil, err
	}
	return
}

// fetchProfile validiates token by using it to fetch user profile
func fetchProfile(t string) (u *user.User, err error) {
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	req, err := http.NewRequest("GET", conf.GLOBUS_PROFILE_URL+"/"+clientId(t), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Globus-Goauthtoken "+t)
	if resp, err := client.Do(req); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			if body, err := ioutil.ReadAll(resp.Body); err == nil {
				u = &user.User{}
				if err = json.Unmarshal(body, &u); err != nil {
					return nil, err
				} else {
					if err = u.SetMongoInfo(); err != nil {
						return nil, err
					}
				}
			}
		} else if resp.StatusCode == http.StatusForbidden {
			return nil, errors.New(e.InvalidAuth)
		} else {
			err_str := "Authentication failed: Unexpected response status: " + resp.Status
			logger.Error(err_str)
			return nil, errors.New(err_str)
		}
	} else {
		return nil, err
	}
	return
}

func clientId(t string) string {
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	req, err := http.NewRequest("GET", conf.GLOBUS_TOKEN_URL, nil)
	if err != nil {
		errStr := "Error creating token request: " + err.Error()
		logger.Error(errStr)
		return ""
	}
	req.Header.Add("X-Globus-Goauthtoken", t)
	if resp, err := client.Do(req); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
			if body, err := ioutil.ReadAll(resp.Body); err == nil {
				var dat map[string]interface{}
				if err = json.Unmarshal(body, &dat); err != nil {
					errStr := "Error unmarshalling JSON body: " + err.Error()
					logger.Error(errStr)
				} else {
					return dat["client_id"].(string)
				}
			}
		} else {
			errStr := "Authentication failed: " + resp.Status
			logger.Error(errStr)
		}
	}
	return ""
}
