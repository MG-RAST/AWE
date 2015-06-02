package goweb

import (
	"errors"
	"net/http"
	"strings"
)

// A handler type to handle actual http requests using the
// DefaultRouteManager to route requests to the right places
type HttpHandler struct {
	routeManager *RouteManager
}

// Serves the HTTP request and writes the response to the specified writer
func (handler *HttpHandler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	var route *Route
	var found bool = false
	var context *Context

	// do we need to spoof the HTTP method?
	overrideMethod := request.URL.Query().Get(REQUEST_METHOD_OVERRIDE_PARAMETER)
	if overrideMethod != "" {
		request.Method = strings.ToUpper(overrideMethod)
	}

	// get the matching route
	found, route, context = handler.GetMathingRoute(responseWriter, request)

	if !found {
		// no route found - this is an error

		// create the request context (with no parameter keys obviously)
		context = makeContext(request, responseWriter, nil)

		error := errors.New(ERR_NO_MATCHING_ROUTE)
		handler.HandleError(context, error)

	} else {
		// tell the controller to handle the route
		var controller Controller = route.Controller

		// make sure we have a controller
		if controller == nil {
			error := errors.New(ERR_NO_CONTROLLER)
			handler.HandleError(context, error)

		} else {
			controller.HandleRequest(context)
		}

	}

}

// Searches DefaultRouteManager to find the first matching route and returns it
// along with a boolean describing whether any routes were found or not, and the
// Context object built while searching for routes
func (h *HttpHandler) GetMathingRoute(responseWriter http.ResponseWriter, request *http.Request) (bool, *Route, *Context) {
	var route *Route
	var found bool = false
	var context *Context
	for i := 0; i < len(h.routeManager.routes); i++ {
		route = h.routeManager.routes[i]
		if route.DoesMatchPath(request.URL.Path) {
			// extract the parameter values
			pathParams := getParameterValueMap(route.parameterKeys, request.URL.Path)

			// create the request context
			context = makeContext(request, responseWriter, pathParams)
			// see if the route matches the context
			if route.DoesMatchContext(context) {
				// found matching route
				found = true
				break
			}
		}

	}
	return found, route, context
}

// Handles the specified error by passing it back to the user
func (h *HttpHandler) HandleError(context *Context, err error) {

	if context.ResponseWriter == nil {
		panic("ResponseWriter cannot be nil")
	}

	// handle the error
	errorString := ERR_STANDARD_PREFIX + err.Error()
	http.Error(context.ResponseWriter, errorString, http.StatusInternalServerError)

}

// The default http handler used to handle requests
var DefaultHttpHandler *HttpHandler = &HttpHandler{routeManager: DefaultRouteManager}

// Listens for incomming requests and handles them using
// the DefaultHttpHandler
//
// The same as:
//
//   http.ListenAndServe(pattern, DefaultHttpHandler)
//
//
// for more information see http.ListenAndServe
//
// A typical example:
//
//   func main() {
//     goweb.Map("/people", peopleController)
//	   goweb.ListenAndServe(":8080")
//   }
//
func ListenAndServe(pattern string) error {
	return http.ListenAndServe(pattern, DefaultHttpHandler)
}

func ListenAndServeTLS(pattern string, certFile string, keyFile string) error {
	return http.ListenAndServeTLS(pattern, certFile, keyFile, DefaultHttpHandler)
}

func ListenAndServeRoutes(pattern string, r *RouteManager) error {
	return http.ListenAndServe(pattern, &HttpHandler{routeManager: r})
}

func ListenAndServeRoutesTLS(pattern string, certFile string, keyFile string, r *RouteManager) error {
	return http.ListenAndServeTLS(pattern, certFile, keyFile, &HttpHandler{routeManager: r})
}
