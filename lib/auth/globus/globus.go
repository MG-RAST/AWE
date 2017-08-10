// Package globus implements Globus Online Nexus authentication
package globus

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/MG-RAST/AWE/lib/auth/basic"
	"github.com/MG-RAST/AWE/lib/conf"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/user"
	"io/ioutil"
	"net/http"
	"strings"
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
	bearer := authHeaderType(header)
	if bearer == "" {
		return nil, errors.New("(globus) Invalid authentication header, missing bearer token.")
	}
	if bearer == "basic" {
		if username, password, err := basic.DecodeHeader(header); err == nil {
			if t, err := fetchToken(username, password); err == nil {
				return fetchProfile(t.AccessToken)
			} else {
				return nil, err
			}
		} else {
			return nil, errors.New("(basic) " + err.Error())
		}
	} else if (bearer == "globus-goauthtoken") || (bearer == "globus") || (bearer == "goauth") {
		// default globus bearer keys
		return fetchProfile(strings.Split(header, " ")[1])
	} else if bearer == "oauth" {
		if conf.OAUTH_DEFAULT == "globus" {
			// allow 'oauth' bearer key if only using globus auth
			return fetchProfile(strings.Split(header, " ")[1])
		} else {
			return nil, errors.New("(globus) Invalid authentication header, unable to use bearer token 'oauth' with multiple oauth servers")
		}
	} else {
		return nil, errors.New("(globus) Invalid authentication header, unknown bearer token: " + bearer)
	}
}

// fetchToken takes username and password and then retrieves user token
func fetchToken(u string, p string) (t *token, err error) {
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	req, err := http.NewRequest("GET", conf.GLOBUS_TOKEN_URL, nil)
	if err != nil {
		return nil, errors.New("(globus) HTTP GET: " + err.Error())
	}
	req.SetBasicAuth(u, p)
	if resp, err := client.Do(req); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			if body, err := ioutil.ReadAll(resp.Body); err == nil {
				if err = json.Unmarshal(body, &t); err != nil {
					return nil, errors.New("(globus) JSON Unmarshal: " + err.Error())
				}
			}
		} else {
			return nil, errors.New("(globus) Authentication failed: Unexpected response status: " + resp.Status)
		}
	} else {
		return nil, errors.New("(globus) " + err.Error())
	}
	return
}

// fetchProfile validiates token by using it to fetch user profile
func fetchProfile(t string) (u *user.User, err error) {
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	cid, err := clientId(t)
	if err != nil {
		return nil, errors.New("(globus) " + err.Error())
	}
	req, err := http.NewRequest("GET", conf.GLOBUS_PROFILE_URL+"/"+cid, nil)
	if err != nil {
		return nil, errors.New("(globus) HTTP GET: " + err.Error())
	}
	req.Header.Add("Authorization", "Globus-Goauthtoken "+t)
	if resp, err := client.Do(req); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			if body, err := ioutil.ReadAll(resp.Body); err == nil {
				u = &user.User{}
				if err = json.Unmarshal(body, &u); err != nil {
					return nil, errors.New("(globus) JSON Unmarshal: " + err.Error())
				} else {
					if u.Username == "" {
						return nil, errors.New("(globus) " + e.InvalidAuth)
					}
					if err = u.SetMongoInfo(); err != nil {
						return nil, errors.New("(globus) MongoDB: " + err.Error())
					}
				}
			}
		} else if resp.StatusCode == http.StatusForbidden {
			return nil, errors.New("(globus) " + e.InvalidAuth)
		} else {
			return nil, errors.New("(globus) Authentication failed: Unexpected response status: " + resp.Status)
		}
	} else {
		return nil, errors.New("(globus) " + err.Error())
	}
	return
}

func clientId(t string) (c string, err error) {
	// test for old format first
	for _, part := range strings.Split(t, "|") {
		if kv := strings.Split(part, "="); kv[0] == "client_id" {
			return kv[1], nil
		}
	}
	//if we get here then we have a new style token and need to make a call to look up the
	//ID instead of parsing the string
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	req, err := http.NewRequest("GET", conf.GLOBUS_TOKEN_URL, nil)
	if err != nil {
		return "", errors.New("(globus) HTTP GET: " + err.Error())
	}
	req.Header.Add("X-Globus-Goauthtoken", t)
	if resp, err := client.Do(req); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
			if body, err := ioutil.ReadAll(resp.Body); err == nil {
				var dat map[string]interface{}
				if err = json.Unmarshal(body, &dat); err != nil {
					return "", errors.New("(globus) JSON Unmarshal: " + err.Error())
				} else {
					return dat["client_id"].(string), nil
				}
			}
		} else if resp.StatusCode == http.StatusForbidden {
			return "", errors.New("(globus) " + e.InvalidAuth)
		} else {
			return "", errors.New("(globus) Authentication failed: Unexpected response status: " + resp.Status)
		}
	} else {
		return "", errors.New("(globus) " + err.Error())
	}
	return
}
