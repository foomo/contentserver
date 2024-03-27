package handler

// Route type
type Route string

const (
	// RouteGetURIs get uris, many at once, to keep it fast
	RouteGetURIs Route = "getURIs"
	// RouteGetContent get (site) content
	RouteGetContent Route = "getContent"
	// RouteGetNodes get nodes
	RouteGetNodes Route = "getNodes"
	// RouteUpdate update repo
	RouteUpdate Route = "update"
	// RouteGetRepo get the whole repo
	RouteGetRepo Route = "getRepo"
)
