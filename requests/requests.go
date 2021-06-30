package requests

// Env - abstract your server state
type Env struct {
	// when resolving conten these are processed in their order
	Dimensions []string `json:"dimensions"`
	// who is it for
	Groups []string `json:"groups"`
}

// Node - an abstract node request, use this one to request navigations
type Node struct {
	// this one should be obvious
	ID string `json:"id"`
	// from which dimension
	Dimension string `json:"dimension"`
	// allowed access groups
	Groups []string `json:"groups"`
	// what do you want to see in your navigations, folders, images or unicorns
	MimeTypes []string `json:"mimeTypes"`
	// expand the navigation tree or just the path to the resolved content
	Expand bool `json:"expand"`
	// Expose hidden nodes
	ExposeHiddenNodes bool `json:"exposeHiddenNodes,omitempty"`
	// filter with these
	DataFields []string `json:"dataFields"`
}

// Nodes - which nodes in which dimensions
type Nodes struct {
	// map[dimension]*node
	Nodes map[string]*Node `json:"nodes"`

	Env *Env `json:"env"`
}

// Content - the standard request to contentserver
type Content struct {
	Env            *Env             `json:"env"`
	URI            string           `json:"URI"`
	Nodes          map[string]*Node `json:"nodes"`
	DataFields     []string         `json:"dataFields"`
	PathDataFields []string         `json:"pathDataFields"`
}

// Update - request an update
type Update struct{}

// Repo - query repo
type Repo struct{}

// ItemMap - map of items
type ItemMap struct {
	ID         string   `json:"id"`
	DataFields []string `json:"dataFields"`
}

// URIs - request multiple URIs for a dimension use this resolve uris for links
// in a document
type URIs struct {
	IDs       []string `json:"ids"`
	Dimension string   `json:"dimension"`
}
