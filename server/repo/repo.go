package repo

import (
	"fmt"
	"github.com/foomo/ContentServer/server/log"
	"github.com/foomo/ContentServer/server/repo/content"
	"github.com/foomo/ContentServer/server/requests"
	"github.com/foomo/ContentServer/server/responses"
	"github.com/foomo/ContentServer/server/utils"
	"strings"
	"time"
)

type Repo struct {
	server       string
	Regions      []string
	Languages    []string
	Directory    map[string]*content.RepoNode
	URIDirectory map[string]*content.RepoNode
	Node         *content.RepoNode
}

func NewRepo(server string) *Repo {
	log.Notice("creating new repo for " + server)
	repo := new(Repo)
	repo.server = server
	return repo
}

func (repo *Repo) ResolveContent(state string, URI string) (resolved bool, resolvedURI string, region string, language string, repoNode *content.RepoNode) {
	parts := strings.Split(URI, content.PATH_SEPARATOR)
	log.Debug("repo.ResolveContent: " + URI)
	for i := len(parts); i > -1; i-- {
		testURI := strings.Join(parts[0:i], content.PATH_SEPARATOR)
		testURIKey := uriKeyForState(state, testURI)
		log.Debug("  testing" + testURIKey)
		if repoNode, ok := repo.URIDirectory[testURIKey]; ok {
			resolved = true
			_, region, language := repoNode.GetLanguageAndRegionForURI(testURI)
			log.Debug("    => " + testURIKey)
			log.Debug("      destination " + fmt.Sprint(repoNode.DestinationIds))
			// check this one
			// repoNode = repo.Directory[repoNode.DestinationIds]

			if languageDestinations, regionOk := repoNode.DestinationIds[region]; regionOk {
				// this check should happen, when updating the repo
				log.Debug("    there is a destionation map for this one " + fmt.Sprint(languageDestinations))
				if languageDestination, destinationOk := languageDestinations[language]; destinationOk {
					if destinationNode, destinationNodeOk := repo.Directory[languageDestination]; destinationNodeOk {
						repoNode = destinationNode
					} else {
						log.Debug("    could not resolve this destinationId : " + languageDestination)
					}
				}
			}
			return true, testURI, region, language, repoNode
		} else {
			log.Debug("    => !" + testURI)
			resolved = false
		}
	}
	return
}

func (repo *Repo) GetItemMap(id string, dataFields []string) map[string]map[string]*content.Item {
	itemMap := make(map[string]map[string]*content.Item)
	if repoNode, ok := repo.Directory[id]; ok {
		for region, languageURIs := range repoNode.URIs {
			itemMap[region] = make(map[string]*content.Item)
			for language, URI := range languageURIs {
				log.Debug(fmt.Sprintf("region :%s language :%s URI: %s", region, language, URI))
				itemMap[region][language] = repoNode.ToItem(region, language, dataFields)
			}
		}
	} else {
		log.Warning("GetItemMapForAllRegionsAndLanguages invalid id " + id)
	}
	return itemMap
}

func (repo *Repo) GetURIs(region string, language string, ids []string) map[string]string {
	uris := make(map[string]string)
	for _, id := range ids {
		uris[id] = repo.GetURI(region, language, id)
	}
	return uris
}

func (repo *Repo) GetURIForNode(region string, language string, repoNode *content.RepoNode) string {

	if repoNode.LinkId == "" {
		languageURIs, regionExists := repoNode.URIs[region]
		if regionExists {
			languageURI, languageURIExists := languageURIs[language]
			if languageURIExists {
				return languageURI
			}
		}
		return ""
	} else {
		return repo.GetURI(region, language, repoNode.LinkId)
	}
}

func (repo *Repo) GetURI(region string, language string, id string) string {
	repoNode, ok := repo.Directory[id]
	if ok {
		return repo.GetURIForNode(region, language, repoNode)
	}
	return ""
}

func (repo *Repo) GetNode(repoNode *content.RepoNode, expanded bool, mimeTypes []string, path []*content.Item, level int, state string, groups []string, region string, language string, dataFields []string) *content.Node {
	node := content.NewNode()
	node.Item = repoNode.ToItem(region, language, dataFields)
	log.Debug("repo.GetNode: " + repoNode.Id)
	for _, childId := range repoNode.Index {
		childNode := repoNode.Nodes[childId]
		if (level == 0 || expanded || !expanded && childNode.InPath(path)) && childNode.InState(state) && !childNode.IsHidden(region, language) && childNode.CanBeAccessedByGroups(groups) && childNode.IsOneOfTheseMimeTypes(mimeTypes) && childNode.InRegion(region) {
			node.Nodes[childId] = repo.GetNode(childNode, expanded, mimeTypes, path, level+1, state, groups, region, language, dataFields)
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
		if treeNode, ok := repo.Directory[nodeRequest.Id]; ok {
			nodes[nodeName] = repo.GetNode(treeNode, nodeRequest.Expand, nodeRequest.MimeTypes, path, 0, r.Env.State, r.Env.Groups, r.Env.Defaults.Region, r.Env.Defaults.Language, nodeRequest.DataFields)
		} else {
			log.Warning("you are requesting an invalid tree node for " + nodeName + " : " + nodeRequest.Id)
		}
	}
	return nodes
}

func (repo *Repo) GetContent(r *requests.Content) *content.SiteContent {
	log.Debug("repo.GetContent: " + r.URI)
	c := content.NewSiteContent()
	resolved, resolvedURI, region, language, node := repo.ResolveContent(r.Env.State, r.URI)
	if resolved {
		log.Notice("200 for " + r.URI)
		// forbidden ?!
		c.Region = region
		c.Language = language
		c.Status = content.STATUS_OK
		c.Handler = node.Handler
		c.URI = resolvedURI
		c.Item = node.ToItem(region, language, []string{})
		c.Path = node.GetPath(region, language)
		c.Data = node.Data
	} else {
		log.Notice("404 for " + r.URI)
		c.Status = content.STATUS_NOT_FOUND
		region = r.Env.Defaults.Region
		language = r.Env.Defaults.Language
	}
	for treeName, treeRequest := range r.Nodes {
		log.Debug("  adding tree " + treeName + " " + treeRequest.Id)
		if treeNode, ok := repo.Directory[treeRequest.Id]; ok {
			c.Nodes[treeName] = repo.GetNode(treeNode, treeRequest.Expand, treeRequest.MimeTypes, c.Path, 0, r.Env.State, r.Env.Groups, region, language, treeRequest.DataFields)
		} else {
			log.Warning("you are requesting an invalid tree node for " + treeName + " : " + treeRequest.Id)
		}
	}
	return c
}

func (repo *Repo) GetRepo() *content.RepoNode {
	return repo.Node
}

func uriKeyForState(state string, uri string) string {
	return state + "-" + uri
}

func builDirectory(dirNode *content.RepoNode, directory map[string]*content.RepoNode, uRIDirectory map[string]*content.RepoNode) {
	log.Debug("repo.buildDirectory: " + dirNode.Id)
	directory[dirNode.Id] = dirNode
	//todo handle duplicate uris
	for _, languageURIs := range dirNode.URIs {
		for _, uri := range languageURIs {
			log.Debug("  uri: " + uri + " => Id: " + dirNode.Id)
			if len(dirNode.States) == 0 {
				uRIDirectory[uriKeyForState("", uri)] = dirNode
			} else {
				for _, state := range dirNode.States {
					uRIDirectory[uriKeyForState(state, uri)] = dirNode
				}
			}

		}
	}
	for _, childNode := range dirNode.Nodes {
		builDirectory(childNode, directory, uRIDirectory)
	}
}

func wireAliases(directory map[string]*content.RepoNode) {
	for _, repoNode := range directory {
		if repoNode.LinkId != "" {
			if destinationNode, ok := directory[repoNode.LinkId]; ok {
				repoNode.URIs = destinationNode.URIs
			}
		}
	}
}

func (repo *Repo) Load(newNode *content.RepoNode) {
	newNode.WireParents()

	newDirectory := make(map[string]*content.RepoNode)
	newURIDirectory := make(map[string]*content.RepoNode)

	builDirectory(newNode, newDirectory, newURIDirectory)
	wireAliases(newDirectory)

	// some more validation anyone?
	//  invalid destination ids

	repo.Node = newNode
	repo.Directory = newDirectory
	repo.URIDirectory = newURIDirectory
}

func (repo *Repo) Update() *responses.Update {
	updateResponse := responses.NewUpdate()

	newNode := content.NewRepoNode()

	startTimeRepo := time.Now()
	ok, err := utils.Get(repo.server, newNode)
	updateResponse.Stats.RepoRuntime = time.Now().Sub(startTimeRepo).Seconds()
	startTimeOwn := time.Now()
	updateResponse.Success = ok
	if ok {
		repo.Load(newNode)
		updateResponse.Stats.NumberOfNodes = len(repo.Directory)
		updateResponse.Stats.NumberOfURIs = len(repo.URIDirectory)
	} else {
		log.Error(fmt.Sprintf("update error: %", err))
		updateResponse.ErrorMessage = fmt.Sprintf("%", err)
		updateResponse.Stats.NumberOfNodes = -1
		updateResponse.Stats.NumberOfURIs = -1
	}
	updateResponse.Stats.OwnRuntime = time.Now().Sub(startTimeOwn).Seconds()
	return updateResponse
}
