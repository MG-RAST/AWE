package goweb

import "net/http"

// The standard API response object
type standardResponse struct {
	S int         `json:"status"`
	D interface{} `json:"data"`
	E []string    `json:"error"`
}

// The standard API response object
type paginatedResponse struct {
	S      int         `json:"status"`
	D      interface{} `json:"data"`
	E      []string    `json:"error"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
	Count  int         `json:"total_count"`
}

// Makes a standardResponse object with the specified settings
func makeStandardResponse() *standardResponse {
	response := new(standardResponse)
	//response.C = ""
	response.S = 200
	response.E = nil
	return response
}

// Makes a standardResponse object with the specified settings
func makePaginatedResponse() *paginatedResponse {
	response := new(paginatedResponse)
	response.S = 200
	response.E = nil
	return response
}

// Makes a successful standardResponse object with the specified settings
func makeSuccessfulStandardResponse(context string, statusCode int, data interface{}) *standardResponse {
	response := makeStandardResponse()
	//response.C = context
	response.S = statusCode
	response.D = data
	return response
}

// Makes an unsuccessful standardResponse object with the specified settings
func makeFailureStandardResponse(context string, statusCode int) *standardResponse {
	response := makeStandardResponse()
	//response.C = context
	response.S = statusCode
	response.E = []string{http.StatusText(statusCode)}
	return response
}
