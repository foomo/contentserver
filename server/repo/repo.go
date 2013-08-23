package repo

import (
	"fmt"
	"github.com/foomo/ContentServer/server/repo/content"
	"github.com/foomo/ContentServer/server/requests"
	"github.com/foomo/ContentServer/server/utils"
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
	repo := new(Repo)
	repo.server = server
	return repo
}

func (repo *Repo) ResolveContent(URI string) (resolved bool, region string, language string, repoNode *content.RepoNode) {
	if repoNode, ok := repo.URIDirectory[URI]; ok {
		resolved = true
		_, region, language := repoNode.GetLanguageAndRegionForURI(URI)
		return true, region, language, repoNode
	} else {
		resolved = false
	}
	return
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

func (repo *Repo) GetNode(repoNode *content.RepoNode, expanded bool, mimeTypes []string, path []string, region string, language string) *content.Node {
	node := content.NewNode()
	node.Item = repoNode.ToItem(region, language)
	for id, childNode := range repoNode.Nodes {
		if expanded && !childNode.Hidden && childNode.IsOneOfTheseMimeTypes(mimeTypes) {
			// || in path and in region mimes
			node.Nodes[id] = repo.GetNode(childNode, expanded, mimeTypes, path, region, language)
		}
	}
	return node
}

func (repo *Repo) GetContent(r *requests.Content) *content.SiteContent {
	c := content.NewSiteContent()
	resolved, region, language, node := repo.ResolveContent(r.URI)
	if resolved {
		// forbidden ?!
		c.Region = region
		c.Language = language
		c.Status = content.STATUS_OK
		c.Item = node.ToItem(region, language)
		c.Path = node.GetPath(region, language)
		for treeName, treeRequest := range r.Nodes {
			// fmt.Println("getting tree", treeName, treeRequest)
			c.Nodes[treeName] = repo.GetNode(repo.Directory[treeRequest.Id], treeRequest.Expand, treeRequest.MimeTypes, []string{}, region, language)
		}
	} else {
		c.Status = content.STATUS_NOT_FOUND
	}
	return c
}

func (repo *Repo) builDirectory(dirNode *content.RepoNode) {
	repo.Directory[dirNode.Id] = dirNode
	for _, languageURIs := range dirNode.URIs {
		for _, URI := range languageURIs {
			fmt.Println(URI, "=>", dirNode.Id)
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
