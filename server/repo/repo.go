package repo

import (
	"errors"
	"fmt"
	"github.com/foomo/contentserver/server/log"
	"github.com/foomo/contentserver/server/repo/content"
	"github.com/foomo/contentserver/server/requests"
	"github.com/foomo/contentserver/server/responses"
	"github.com/foomo/contentserver/server/utils"
	golog "log"
	"strings"
	"time"
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
}

func NewRepo(server string) *Repo {
	log.Notice("creating new repo for " + server)
	repo := new(Repo)
	repo.Directory = make(map[string]*Dimension)
	repo.server = server
	repo.updateChannel = make(chan *RepoDimension)
	repo.updateDoneChannel = make(chan error)
	go func() {
		for {
			select {
			case newDimension := <-repo.updateChannel:
				repo.updateDoneChannel <- repo.load(newDimension.Dimension, newDimension.Node)
			}
		}
	}()
	return repo
}

func (repo *Repo) Load(dimension string, node *content.RepoNode) error {
	repo.updateChannel <- &RepoDimension{
		Dimension: dimension,
		Node:      node,
	}
	return <-repo.updateDoneChannel
}

func (repo *Repo) load(dimension string, newNode *content.RepoNode) error {
	newNode.WireParents()
	newDirectory := make(map[string]*content.RepoNode)
	newURIDirectory := make(map[string]*content.RepoNode)

	err := builDirectory(newNode, newDirectory, newURIDirectory)
	if err != nil {
		return err
	}
	err = wireAliases(newDirectory)
	if err != nil {
		return err
	}
	repo.Directory[dimension] = &Dimension{
		Node:         newNode,
		Directory:    newDirectory,
		URIDirectory: newURIDirectory,
	}
	return nil
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
	} else {
		log.Notice("404 for " + r.URI)
		c.Status = content.STATUS_NOT_FOUND
		c.Dimension = r.Env.Dimensions[0]
	}
	log.Debug(fmt.Sprintf("resolved: %v, uri: %v, dim: %v, n: %v", resolved, resolvedURI, resolvedDimension, node))
	if resolved == false {
		resolvedDimension = r.Env.Dimensions[0]
	}
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

func builDirectory(dirNode *content.RepoNode, directory map[string]*content.RepoNode, uRIDirectory map[string]*content.RepoNode) error {
	log.Debug("repo.buildDirectory: " + dirNode.Id)
	existingNode, ok := directory[dirNode.Id]
	if ok {
		return errors.New("duplicate node with id:" + existingNode.Id)
	}
	directory[dirNode.Id] = dirNode
	//todo handle duplicate uris
	if _, thereIsAnExistingUriNode := uRIDirectory[dirNode.URI]; thereIsAnExistingUriNode {
		return errors.New("duplicate node with uri: " + dirNode.URI)
	}
	uRIDirectory[dirNode.URI] = dirNode
	for _, childNode := range dirNode.Nodes {
		err := builDirectory(childNode, directory, uRIDirectory)
		if err != nil {
			return err
		}
	}
	return nil
}

func wireAliases(directory map[string]*content.RepoNode) error {
	for _, repoNode := range directory {
		if len(repoNode.LinkId) > 0 {
			if destinationNode, ok := directory[repoNode.LinkId]; ok {
				repoNode.URI = destinationNode.URI
			} else {
				return errors.New("that link id points nowhere " + repoNode.LinkId + " from " + repoNode.Id)
			}
		}
	}
	return nil
}

func (repo *Repo) Update() *responses.Update {
	updateResponse := responses.NewUpdate()
	newNodes := make(map[string]*content.RepoNode) //content.NewRepoNode()
	startTimeRepo := time.Now()
	ok, err := utils.GetRepo(repo.server, newNodes)
	updateResponse.Stats.RepoRuntime = time.Now().Sub(startTimeRepo).Seconds()
	startTimeOwn := time.Now()
	updateResponse.Success = ok
	if ok {
		log.Debug("going to load dimensions from" + utils.ToJSON(newNodes))
		for dimension, newNode := range newNodes {
			log.Debug("loading nodes for dimension " + dimension)
			loadErr := repo.Load(dimension, newNode)
			if loadErr != nil {
				golog.Println(loadErr)
				panic(loadErr)
			}
			log.Debug("loaded nodes for dimension " + dimension)
			_, dimensionOk := repo.Directory[dimension]
			if dimensionOk {
				updateResponse.Stats.NumberOfNodes += len(repo.Directory[dimension].Directory)
				updateResponse.Stats.NumberOfURIs += len(repo.Directory[dimension].URIDirectory)
			} else {
				log.Debug("where is dimension " + dimension)
				golog.Println(repo.Directory)
			}
		}
	} else {
		log.Error(fmt.Sprintf("update error: %", err))
		updateResponse.ErrorMessage = fmt.Sprintf("%", err)
		updateResponse.Stats.NumberOfNodes = -1
		updateResponse.Stats.NumberOfURIs = -1
	}
	updateResponse.Stats.OwnRuntime = time.Now().Sub(startTimeOwn).Seconds()
	return updateResponse
}
