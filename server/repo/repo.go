package repo

import (
	"fmt"
	"strings"

	"github.com/foomo/contentserver/server/log"
	"github.com/foomo/contentserver/server/repo/content"
	"github.com/foomo/contentserver/server/requests"
)

type Dimension struct {
	Directory    map[string]*content.RepoNode
	URIDirectory map[string]*content.RepoNode
	Node         *content.RepoNode
}

type RepoDimension struct {
	Dimension string
	Node      *content.RepoNode
}

type Repo struct {
	server            string
	Directory         map[string]*Dimension
	updateChannel     chan *RepoDimension
	updateDoneChannel chan error
	history           *history
}

func NewRepo(server string, varDir string) *Repo {
	log.Notice("creating new repo for " + server)
	log.Notice("	using var dir:" + varDir)
	repo := new(Repo)
	repo.Directory = make(map[string]*Dimension)
	repo.server = server
	repo.history = newHistory(varDir)
	repo.updateChannel = make(chan *RepoDimension)
	repo.updateDoneChannel = make(chan error)
	go repo.updateRoutine()
	log.Record("trying to restore pervious state")
	restoreErr := repo.tryToRestoreCurrent()
	if restoreErr != nil {
		log.Record("	could not restore previous repo content:" + restoreErr.Error())
	} else {
		log.Record("	restored previous repo content")
	}
	return repo
}

func (repo *Repo) ResolveContent(dimensions []string, URI string) (resolved bool, resolvedURI string, resolvedDimension string, repoNode *content.RepoNode) {
	parts := strings.Split(URI, content.PATH_SEPARATOR)
	resolved = false
	resolvedURI = ""
	resolvedDimension = ""
	repoNode = nil
	log.Debug("repo.ResolveContent: " + URI)
	for _, dimension := range dimensions {
		if d, ok := repo.Directory[dimension]; ok {
			for i := len(parts); i > 0; i-- {
				testURI := strings.Join(parts[0:i], content.PATH_SEPARATOR)
				if testURI == "" {
					testURI = content.PATH_SEPARATOR
				}
				log.Debug("  testing[" + dimension + "]: " + testURI)
				if repoNode, ok := d.URIDirectory[testURI]; ok {
					resolved = true
					log.Debug("  found  => " + testURI)
					log.Debug("    destination " + fmt.Sprint(repoNode.DestinationId))
					if len(repoNode.DestinationId) > 0 {
						if destionationNode, destinationNodeOk := d.Directory[repoNode.DestinationId]; destinationNodeOk {
							repoNode = destionationNode
						}
					}
					return true, testURI, dimension, repoNode
				}
			}
		}
	}
	return
}

func (repo *Repo) GetURIs(dimension string, ids []string) map[string]string {
	uris := make(map[string]string)
	for _, id := range ids {
		uris[id] = repo.GetURI(dimension, id)
	}
	return uris
}

func (repo *Repo) GetURIForNode(dimension string, repoNode *content.RepoNode) string {

	if len(repoNode.LinkId) == 0 {
		return repoNode.URI
	} else {
		linkedNode, ok := repo.Directory[dimension].Directory[repoNode.LinkId]
		if ok {
			return repo.GetURIForNode(dimension, linkedNode)
		} else {
			return ""
		}
	}
}

func (repo *Repo) GetURI(dimension string, id string) string {
	repoNode, ok := repo.Directory[dimension].Directory[id]
	if ok {
		return repo.GetURIForNode(dimension, repoNode)
	}
	return ""
}

func (repo *Repo) GetNode(repoNode *content.RepoNode, expanded bool, mimeTypes []string, path []*content.Item, level int, groups []string, dataFields []string) *content.Node {
	node := content.NewNode()
	node.Item = repoNode.ToItem(dataFields)
	log.Debug("repo.GetNode: " + repoNode.Id)
	for _, childId := range repoNode.Index {
		childNode := repoNode.Nodes[childId]
		if (level == 0 || expanded || !expanded && childNode.InPath(path)) && !childNode.Hidden && childNode.CanBeAccessedByGroups(groups) && childNode.IsOneOfTheseMimeTypes(mimeTypes) {
			node.Nodes[childId] = repo.GetNode(childNode, expanded, mimeTypes, path, level+1, groups, dataFields)
			node.Index = append(node.Index, childId)
		}
	}
	return node
}

func (repo *Repo) GetNodes(r *requests.Nodes) map[string]*content.Node {
	nodes := make(map[string]*content.Node)
	path := make([]*content.Item, 0)
	for nodeName, nodeRequest := range r.Nodes {
		log.Debug("  adding node " + nodeName + " " + nodeRequest.Id)
		if treeNode, ok := repo.Directory[nodeRequest.Dimension].Directory[nodeRequest.Id]; ok {
			nodes[nodeName] = repo.GetNode(treeNode, nodeRequest.Expand, nodeRequest.MimeTypes, path, 0, r.Env.Groups, nodeRequest.DataFields)
		} else {
			log.Warning("you are requesting an invalid tree node for " + nodeName + " : " + nodeRequest.Id)
		}
	}
	return nodes
}

func (repo *Repo) GetContent(r *requests.Content) *content.SiteContent {
	// add more input validation
	log.Debug("repo.GetContent: " + r.URI)
	c := content.NewSiteContent()
	resolved, resolvedURI, resolvedDimension, node := repo.ResolveContent(r.Env.Dimensions, r.URI)
	if resolved {
		log.Notice("200 for " + r.URI)
		// forbidden ?!
		c.Status = content.STATUS_OK
		c.MimeType = node.MimeType
		c.Dimension = resolvedDimension
		c.URI = resolvedURI
		c.Item = node.ToItem([]string{})
		c.Path = node.GetPath()
		c.Data = node.Data
		// fetch URIs for all dimensions
		uris := make(map[string]string)
		for dimensionName, _ := range repo.Directory {
			uris[dimensionName] = repo.GetURI(dimensionName, node.Id)
		}
		c.URIs = uris
	} else {
		log.Notice("404 for " + r.URI)
		c.Status = content.STATUS_NOT_FOUND
		c.Dimension = r.Env.Dimensions[0]
	}
	log.Debug(fmt.Sprintf("resolved: %v, uri: %v, dim: %v, n: %v", resolved, resolvedURI, resolvedDimension, node))
	if resolved == false {
		resolvedDimension = r.Env.Dimensions[0]
	}
	// add navigation trees
	for treeName, treeRequest := range r.Nodes {
		log.Debug("  adding tree " + treeName + " " + treeRequest.Id)
		if treeNode, ok := repo.Directory[resolvedDimension].Directory[treeRequest.Id]; ok {
			c.Nodes[treeName] = repo.GetNode(treeNode, treeRequest.Expand, treeRequest.MimeTypes, c.Path, 0, r.Env.Groups, treeRequest.DataFields)
		} else {
			log.Warning("you are requesting an invalid tree node for " + treeName + " : " + treeRequest.Id)
		}
	}
	return c
}

func (repo *Repo) GetRepo() map[string]*content.RepoNode {
	response := make(map[string]*content.RepoNode)
	for dimensionName, dimension := range repo.Directory {
		response[dimensionName] = dimension.Node
	}
	return response
}

func uriKeyForState(state string, uri string) string {
	return state + "-" + uri
}
