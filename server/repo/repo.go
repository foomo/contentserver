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

func (repo *Repo) ResolveContent(URI string) (resolved bool, resolvedURI string, region string, language string, repoNode *content.RepoNode) {
	parts := strings.Split(URI, content.PATH_SEPARATOR)
	log.Debug("repo.ResolveContent: " + URI)
	for i := len(parts); i > -1; i-- {
		testURI := strings.Join(parts[0:i], content.PATH_SEPARATOR)
		log.Debug("  testing" + testURI)
		if repoNode, ok := repo.URIDirectory[testURI]; ok {
			resolved = true
			_, region, language := repoNode.GetLanguageAndRegionForURI(testURI)
			log.Debug("    => " + testURI)
			log.Debug("      destionations " + fmt.Sprint(repoNode.DestinationIds))
			log.Debug("        .region " + region + "  " + fmt.Sprint(repoNode.DestinationIds[region]))
			if languageDestinations, regionOk := repoNode.DestinationIds[region]; regionOk {
				log.Debug("    there is a destionation map for this one " + fmt.Sprint(languageDestinations))
				if languageDestination, destinationOk := languageDestinations[language]; destinationOk {
					// what if it is not there ....
					repoNode = repo.Directory[languageDestination]
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

func (repo *Repo) GetItemMap(id string) map[string]map[string]*content.Item {
	itemMap := make(map[string]map[string]*content.Item)
	if repoNode, ok := repo.Directory[id]; ok {
		for region, languageURIs := range repoNode.URIs {
			itemMap[region] = make(map[string]*content.Item)
			for language, URI := range languageURIs {
				log.Debug(fmt.Sprintf("region :%s language :%s URI: %s", region, language, URI))
				itemMap[region][language] = repoNode.ToItem(region, language)
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

func (repo *Repo) GetURI(region string, language string, id string) string {
	repoNode, ok := repo.Directory[id]
	if ok {
		languageURIs, regionExists := repoNode.URIs[region]
		if regionExists {
			languageURI, languageURIExists := languageURIs[language]
			if languageURIExists {
				return languageURI
			}
		}
	}
	return ""
}

func (repo *Repo) GetNode(repoNode *content.RepoNode, expanded bool, mimeTypes []string, path []*content.Item, level int, groups []string, region string, language string) *content.Node {
	node := content.NewNode()
	node.Item = repoNode.ToItem(region, language)
	log.Debug("repo.GetNode: " + repoNode.Id)
	for _, childId := range repoNode.Index {
		childNode := repoNode.Nodes[childId]
		if (level == 0 || expanded || !expanded && childNode.InPath(path)) && !childNode.Hidden && childNode.CanBeAccessedByGroups(groups) && childNode.IsOneOfTheseMimeTypes(mimeTypes) && childNode.InRegion(region) {
			node.Nodes[childId] = repo.GetNode(childNode, expanded, mimeTypes, path, level+1, groups, region, language)
			node.Index = append(node.Index, childId)
		}
		// fmt.Println("no see for", childNode.GetName(region, language), childNode.Hidden)

	}
	return node
}

func (repo *Repo) GetNodes(r *requests.Nodes) map[string]*content.Node {
	nodes := make(map[string]*content.Node)
	path := make([]*content.Item, 0)
	for nodeName, nodeRequest := range r.Nodes {
		log.Debug("  adding node " + nodeName + " " + nodeRequest.Id)
		if treeNode, ok := repo.Directory[nodeRequest.Id]; ok {
			nodes[nodeName] = repo.GetNode(treeNode, nodeRequest.Expand, nodeRequest.MimeTypes, path, 0, r.Env.Groups, r.Env.Defaults.Region, r.Env.Defaults.Language)
		} else {
			log.Warning("you are requesting an invalid tree node for " + nodeName + " : " + nodeRequest.Id)
		}
	}
	return nodes
}

func (repo *Repo) GetContent(r *requests.Content) *content.SiteContent {
	log.Debug("repo.GetContent: " + r.URI)
	c := content.NewSiteContent()
	resolved, resolvedURI, region, language, node := repo.ResolveContent(r.URI)
	if resolved {
		log.Notice("200 for " + r.URI)
		// forbidden ?!
		c.Region = region
		c.Language = language
		c.Status = content.STATUS_OK
		c.Handler = node.Handler
		c.URI = resolvedURI
		c.Item = node.ToItem(region, language)
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
			c.Nodes[treeName] = repo.GetNode(treeNode, treeRequest.Expand, treeRequest.MimeTypes, c.Path, 0, r.Env.Groups, region, language)
		} else {
			log.Warning("you are requesting an invalid tree node for " + treeName + " : " + treeRequest.Id)
		}
	}
	return c
}

func builDirectory(dirNode *content.RepoNode, directory map[string]*content.RepoNode, uRIDirectory map[string]*content.RepoNode) {
	log.Debug("repo.buildDirectory: " + dirNode.Id)
	directory[dirNode.Id] = dirNode
	//todo handle duplicate uris
	for _, languageURIs := range dirNode.URIs {
		for _, URI := range languageURIs {
			log.Debug("  URI: " + URI + " => Id: " + dirNode.Id)
			uRIDirectory[URI] = dirNode
		}
	}
	for _, childNode := range dirNode.Nodes {
		builDirectory(childNode, directory, uRIDirectory)
	}
}

func (repo *Repo) Update() *responses.Update {
	updateResponse := responses.NewUpdate()

	newNode := content.NewRepoNode()

	startTimeRepo := time.Now()
	utils.Get(repo.server, newNode)
	updateResponse.Stats.RepoRuntime = time.Now().Sub(startTimeRepo).Seconds()

	startTimeOwn := time.Now()
	newNode.WireParents()

	newDirectory := make(map[string]*content.RepoNode)
	newURIDirectory := make(map[string]*content.RepoNode)

	builDirectory(newNode, newDirectory, newURIDirectory)

	repo.Node = newNode
	repo.Directory = newDirectory
	repo.URIDirectory = newURIDirectory

	updateResponse.Stats.OwnRuntime = time.Now().Sub(startTimeOwn).Seconds()

	updateResponse.Stats.NumberOfNodes = len(repo.Directory)
	updateResponse.Stats.NumberOfURIs = len(repo.URIDirectory)

	return updateResponse
}
