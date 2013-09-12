package repo

import (
	"fmt"
	"github.com/foomo/ContentServer/server/log"
	"github.com/foomo/ContentServer/server/repo/content"
	"github.com/foomo/ContentServer/server/requests"
	"github.com/foomo/ContentServer/server/utils"
	"strings"
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

func (repo *Repo) GetNode(repoNode *content.RepoNode, expanded bool, mimeTypes []string, path []*content.Item, region string, language string) *content.Node {
	node := content.NewNode()
	node.Item = repoNode.ToItem(region, language)
	log.Debug("repo.GetNode: " + repoNode.Id)
	for _, childId := range repoNode.Index {
		childNode := repoNode.Nodes[childId]
		if (expanded || !expanded && childNode.InPath(path)) && !childNode.Hidden && childNode.IsOneOfTheseMimeTypes(mimeTypes) && childNode.InRegion(region) {
			node.Nodes[childId] = repo.GetNode(childNode, expanded, mimeTypes, path, region, language)
			node.Index = append(node.Index, childId)
		} else {
			fmt.Println("no see for", childNode.GetName(language), childNode.Hidden)
		}

	}
	return node
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
		for treeName, treeRequest := range r.Nodes {
			log.Debug("  adding tree " + treeName + " " + treeRequest.Id)
			c.Nodes[treeName] = repo.GetNode(repo.Directory[treeRequest.Id], treeRequest.Expand, treeRequest.MimeTypes, c.Path, region, language)
		}
	} else {
		log.Notice("404 for " + r.URI)
		c.Status = content.STATUS_NOT_FOUND
	}
	return c
}

func (repo *Repo) builDirectory(dirNode *content.RepoNode) {
	log.Debug("repo.buildDirectory: " + dirNode.Id)
	repo.Directory[dirNode.Id] = dirNode
	//todo handle duplicate uris
	for _, languageURIs := range dirNode.URIs {
		for _, URI := range languageURIs {
			log.Debug("  URI: " + URI + " => Id: " + dirNode.Id)
			repo.URIDirectory[URI] = dirNode
		}
	}
	for _, childNode := range dirNode.Nodes {
		repo.builDirectory(childNode)
	}
}

func (repo *Repo) Update() {
	repo.Node = content.NewRepoNode()
	utils.Get(repo.server, repo.Node)
	repo.Node.WireParents()
	repo.Directory = make(map[string]*content.RepoNode)
	repo.URIDirectory = make(map[string]*content.RepoNode)
	repo.builDirectory(repo.Node)
}
