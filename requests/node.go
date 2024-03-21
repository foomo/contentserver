package requests

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
