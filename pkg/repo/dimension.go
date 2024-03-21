package repo

import (
	"github.com/foomo/contentserver/content"
)

// Dimension dimension in a repo
type Dimension struct {
	Directory    map[string]*content.RepoNode
	URIDirectory map[string]*content.RepoNode
	Node         *content.RepoNode
}
