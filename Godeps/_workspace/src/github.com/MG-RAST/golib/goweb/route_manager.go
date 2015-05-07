package goweb

/*
	RouteManager
*/

// Manages routes and matching
type RouteManager struct {
	routes []*Route
}

// Creates a route that maps the specified path to the specified controller
// along with any optional RouteMatcherFunc modifiers
func (manager *RouteManager) Map(path string, controller Controller, matcherFuncs ...RouteMatcherFunc) *Route {

	// create the route (from the path)
	route := makeRouteFromPath(path)

	// set the controller
	route.Controller = controller

	// set the matcher funcs
	route.MatcherFuncs = matcherFuncs

	// add the new route to the default
	manager.AddRoute(route)

	// return the new route
	return route

}

func (manager *RouteManager) MapFunc(path string, contorllerFunction func(*Context), matcherFuncs ...RouteMatcherFunc) *Route {

	// create the route (from the path)
	route := makeRouteFromPath(path)

	// set the controller
	route.Controller = ControllerFunc(contorllerFunction)

	// set the matcher funcs
	route.MatcherFuncs = matcherFuncs

	// add the new route to the default
	manager.AddRoute(route)

	// return the new route
	return route

}

func (manager *RouteManager) MapRest(pathPrefix string, controller RestController) {

	var pathPrefixWithId string = pathPrefix + "/{id}"

	// OPTIONS /resource
	if rc, ok := controller.(RestOptions); ok {
		manager.MapFunc(pathPrefix, func(c *Context) {
			rc.Options(c)
		}, OptionsMethod)
	}
	// GET /resource/{id}
	if rc, ok := controller.(RestReader); ok {
		manager.MapFunc(pathPrefixWithId, func(c *Context) {
			rc.Read(c.PathParams["id"], c)
		}, GetMethod)
	}
	// GET /resource
	if rc, ok := controller.(RestManyReader); ok {
		manager.MapFunc(pathPrefix, func(c *Context) {
			rc.ReadMany(c)
		}, GetMethod)
	}
	// PUT /resource/{id}
	if rc, ok := controller.(RestUpdater); ok {
		manager.MapFunc(pathPrefixWithId, func(c *Context) {
			rc.Update(c.PathParams["id"], c)
		}, PutMethod)
	}
	// PUT /resource
	if rc, ok := controller.(RestManyUpdater); ok {
		manager.MapFunc(pathPrefix, func(c *Context) {
			rc.UpdateMany(c)
		}, PutMethod)
	}
	// DELETE /resource/{id}
	if rc, ok := controller.(RestDeleter); ok {
		manager.MapFunc(pathPrefixWithId, func(c *Context) {
			rc.Delete(c.PathParams["id"], c)
		}, DeleteMethod)
	}
	// DELETE /resource
	if rc, ok := controller.(RestManyDeleter); ok {
		manager.MapFunc(pathPrefix, func(c *Context) {
			rc.DeleteMany(c)
		}, DeleteMethod)
	}
	// CREATE /resource/{id}
	if rc, ok := controller.(RestCreatorWithId); ok {
		manager.MapFunc(pathPrefixWithId, func(c *Context) {
			rc.CreateWithId(c.PathParams["id"], c)
		}, PostMethod)
	}
	// CREATE /resource
	if rc, ok := controller.(RestCreator); ok {
		manager.MapFunc(pathPrefix, func(c *Context) {
			rc.Create(c)
		}, PostMethod)
	}
}

// Adds a route to the manager
func (manager *RouteManager) AddRoute(route *Route) {
	manager.routes = append(manager.routes, route)
}

// Clears all routes
func (manager *RouteManager) ClearRoutes() {
	manager.routes = make([]*Route, 0)
}

// Default instance of the RouteManager
var DefaultRouteManager *RouteManager = new(RouteManager)
