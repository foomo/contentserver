package repo

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/log"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
)

// Dimension dimension in a repo
type Dimension struct {
	Directory    map[string]*content.RepoNode
	URIDirectory map[string]*content.RepoNode
	Node         *content.RepoNode
}

// Repo content repositiory
type Repo struct {
	server            string
	Directory         map[string]*Dimension
	updateChannel     chan *repoDimension
	updateDoneChannel chan error
	history           *history
}

type repoDimension struct {
	Dimension string
	Node      *content.RepoNode
}

// NewRepo constructor
func NewRepo(server string, varDir string) *Repo {
	log.Notice("creating new repo for " + server)
	log.Notice("	using var dir:" + varDir)
	repo := &Repo{
		server:            server,
		Directory:         map[string]*Dimension{},
		history:           newHistory(varDir),
		updateChannel:     make(chan *repoDimension),
		updateDoneChannel: make(chan error),
	}
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

// GetURIs get many uris at once
func (repo *Repo) GetURIs(dimension string, ids []string) map[string]string {
	uris := map[string]string{}
	for _, id := range ids {
		uris[id] = repo.getURI(dimension, id)
	}
	return uris
}

// GetNodes get nodes
func (repo *Repo) GetNodes(r *requests.Nodes) map[string]*content.Node {
	return repo.getNodes(r.Nodes, r.Env.Groups)
}

func (repo *Repo) getNodes(nodeRequests map[string]*requests.Node, groups []string) map[string]*content.Node {
	nodes := map[string]*content.Node{}
	path := []*content.Item{}
	for nodeName, nodeRequest := range nodeRequests {
		log.Debug("  adding node " + nodeName + " " + nodeRequest.ID)
		dimensionNode, ok := repo.Directory[nodeRequest.Dimension]
		nodes[nodeName] = nil
		if !ok {
			log.Warning("could not get dimension root node for nodeRequest.Dimension: " + nodeRequest.Dimension)
			continue
		}
		treeNode, ok := dimensionNode.Directory[nodeRequest.ID]
		if ok {
			nodes[nodeName] = repo.getNode(treeNode, nodeRequest.Expand, nodeRequest.MimeTypes, path, 0, groups, nodeRequest.DataFields)
		} else {
			log.Warning("you are requesting an invalid tree node for " + nodeName + " : " + nodeRequest.ID)
		}
	}
	return nodes
}

// GetContent resolves content and fetches nodes in one call. It combines those
// two tasks for performance reasons.
//
// In the first step it uses r.URI to look up content in all given
// r.Env.Dimensions of repo.Directory.
//
// In the second step it collects the requested nodes.
//
// those two steps are independent.
func (repo *Repo) GetContent(r *requests.Content) (c *content.SiteContent, err error) {
	// add more input validation
	err = repo.validateContentRequest(r)
	if err != nil {
		log.Debug("repo.GetContent invalid request", err)
		return
	}
	log.Debug("repo.GetContent: ", r.URI)
	c = content.NewSiteContent()
	resolved, resolvedURI, resolvedDimension, node := repo.resolveContent(r.Env.Dimensions, r.URI)
	if resolved {
		log.Notice("200 for " + r.URI)
		// forbidden ?!
		c.Status = content.StatusOk
		c.MimeType = node.MimeType
		c.Dimension = resolvedDimension
		c.URI = resolvedURI
		c.Item = node.ToItem([]string{})
		c.Path = node.GetPath()
		c.Data = node.Data
		// fetch URIs for all dimensions
		uris := make(map[string]string)
		for dimensionName := range repo.Directory {
			uris[dimensionName] = repo.getURI(dimensionName, node.ID)
		}
		c.URIs = uris
	} else {
		log.Notice("404 for " + r.URI)
		c.Status = content.StatusNotFound
		c.Dimension = r.Env.Dimensions[0]
	}
	if log.SelectedLevel == log.LevelDebug {
		log.Debug(fmt.Sprintf("resolved: %v, uri: %v, dim: %v, n: %v", resolved, resolvedURI, resolvedDimension, node))
	}
	if resolved == false {
		log.Debug("repo.GetContent", r.URI, "could not be resolved falling back to default dimension", r.Env.Dimensions[0])
		// r.Env.Dimensions is validated => we can access it
		resolvedDimension = r.Env.Dimensions[0]
	}
	// add navigation trees
	for _, node := range r.Nodes {
		if node.Dimension == "" {
			node.Dimension = resolvedDimension
		}
	}
	c.Nodes = repo.getNodes(r.Nodes, r.Env.Groups)
	return c, nil
}

// GetRepo get the whole repo in all dimensions
func (repo *Repo) GetRepo() map[string]*content.RepoNode {
	response := make(map[string]*content.RepoNode)
	for dimensionName, dimension := range repo.Directory {
		response[dimensionName] = dimension.Node
	}
	return response
}

// Update - reload contents of repository with json from repo.server
func (repo *Repo) Update() (updateResponse *responses.Update) {
	floatSeconds := func(nanoSeconds int64) float64 {
		return float64(float64(nanoSeconds) / float64(1000000000.0))
	}
	startTime := time.Now().UnixNano()
	updateRepotime, jsonBytes, updateErr := repo.update()
	updateResponse = &responses.Update{}
	updateResponse.Stats.RepoRuntime = floatSeconds(updateRepotime)

	if updateErr != nil {
		updateResponse.Success = false
		updateResponse.Stats.NumberOfNodes = -1
		updateResponse.Stats.NumberOfURIs = -1
		// let us try to restore the world from a file
		log.Error("could not update repository:" + updateErr.Error())
		updateResponse.ErrorMessage = updateErr.Error()
		restoreErr := repo.tryToRestoreCurrent()
		if restoreErr != nil {
			log.Error("failed to restore preceding repo version: " + restoreErr.Error())
		} else {
			log.Record("restored current repo from local history")
		}
	} else {
		updateResponse.Success = true
		// persist the currently loaded one
		historyErr := repo.history.add(jsonBytes)
		if historyErr != nil {
			log.Warning("could not persist current repo in history: " + historyErr.Error())
		}
		// add some stats
		for dimension := range repo.Directory {
			updateResponse.Stats.NumberOfNodes += len(repo.Directory[dimension].Directory)
			updateResponse.Stats.NumberOfURIs += len(repo.Directory[dimension].URIDirectory)
		}
	}
	updateResponse.Stats.OwnRuntime = floatSeconds(time.Now().UnixNano()-startTime) - updateResponse.Stats.RepoRuntime
	return updateResponse
}

// resolveContent find content in a repository
func (repo *Repo) resolveContent(dimensions []string, URI string) (resolved bool, resolvedURI string, resolvedDimension string, repoNode *content.RepoNode) {
	parts := strings.Split(URI, content.PathSeparator)
	resolved = false
	resolvedURI = ""
	resolvedDimension = ""
	repoNode = nil
	log.Debug("repo.ResolveContent: " + URI)
	for _, dimension := range dimensions {
		if d, ok := repo.Directory[dimension]; ok {
			for i := len(parts); i > 0; i-- {
				testURI := strings.Join(parts[0:i], content.PathSeparator)
				if testURI == "" {
					testURI = content.PathSeparator
				}
				log.Debug("  testing[" + dimension + "]: " + testURI)
				if repoNode, ok := d.URIDirectory[testURI]; ok {
					resolved = true
					log.Debug("  found  => " + testURI)
					log.Debug("    destination " + fmt.Sprint(repoNode.DestinationID))
					if len(repoNode.DestinationID) > 0 {
						if destionationNode, destinationNodeOk := d.Directory[repoNode.DestinationID]; destinationNodeOk {
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

func (repo *Repo) getURIForNode(dimension string, repoNode *content.RepoNode) string {

	if len(repoNode.LinkID) == 0 {
		return repoNode.URI
	}
	linkedNode, ok := repo.Directory[dimension].Directory[repoNode.LinkID]
	if ok {
		return repo.getURIForNode(dimension, linkedNode)
	}
	return ""
}

func (repo *Repo) getURI(dimension string, id string) string {
	repoNode, ok := repo.Directory[dimension].Directory[id]
	if ok {
		return repo.getURIForNode(dimension, repoNode)
	}
	return ""
}

func (repo *Repo) getNode(repoNode *content.RepoNode, expanded bool, mimeTypes []string, path []*content.Item, level int, groups []string, dataFields []string) *content.Node {
	node := content.NewNode()
	node.Item = repoNode.ToItem(dataFields)
	log.Debug("repo.GetNode: " + repoNode.ID)
	for _, childID := range repoNode.Index {
		childNode := repoNode.Nodes[childID]
		if (level == 0 || expanded || !expanded && childNode.InPath(path)) && !childNode.Hidden && childNode.CanBeAccessedByGroups(groups) && childNode.IsOneOfTheseMimeTypes(mimeTypes) {
			node.Nodes[childID] = repo.getNode(childNode, expanded, mimeTypes, path, level+1, groups, dataFields)
			node.Index = append(node.Index, childID)
		}
	}
	return node
}

func (repo *Repo) validateContentRequest(req *requests.Content) (err error) {
	if req == nil {
		return errors.New("request must not be nil")
	}
	if len(req.URI) == 0 {
		return errors.New("request URI must not be empty")
	}
	if req.Env == nil {
		return errors.New("request.Env must not be nil")
	}
	if len(req.Env.Dimensions) == 0 {
		return errors.New("request.Env.Dimensions must not be empty")
	}
	for _, envDimension := range req.Env.Dimensions {
		if !repo.hasDimension(envDimension) {
			availableDimensions := []string{}
			for availableDimension := range repo.Directory {
				availableDimensions = append(availableDimensions, availableDimension)
			}
			return fmt.Errorf("unknown dimension %q in r.Env must be one of %q", envDimension, availableDimensions)
		}
	}
	return nil
}

func (repo *Repo) hasDimension(d string) bool {
	_, hasDimension := repo.Directory[d]
	return hasDimension
}

func uriKeyForState(state string, uri string) string {
	return state + "-" + uri
}
