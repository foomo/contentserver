package repo

import (
	"fmt"
	"github.com/foomo/ContentServer/server/repo/content"
	"github.com/foomo/ContentServer/server/requests"
	"github.com/foomo/ContentServer/server/utils"
)

type Repo struct {
	server    string
	Regions   []string
	Languages []string
	Directory map[string]content.RepoNode
	Node      *content.RepoNode
}

func NewRepo(server string) *Repo {
	repo := new(Repo)
	repo.server = server
	return repo
}

func (repo *Repo) GetContent(r *requests.Content) *content.SiteContent {
	c := content.NewSiteContent()
	if node, ok := repo.Directory[r.URI]; ok {
		c.Status = content.STATUS_OK
		fmt.Println(node.Names, r.Env.Language)
		c.Content.Item = content.NewItem()
		c.Content.Item.Id = node.Id
		c.Content.Item.Name = node.GetName(r.Env.Language)
	} else {
		c.Status = content.STATUS_NOT_FOUND
	}
	return c
}

func (repo *Repo) builDirectory(dirNode *content.RepoNode) {
	repo.Directory[dirNode.Id] = *dirNode
	for _, childNode := range dirNode.Nodes {
		repo.builDirectory(childNode)
	}
}

func (repo *Repo) Update() {
	repo.Node = content.NewRepoNode()
	utils.Get(repo.server, repo.Node)
	repo.Directory = make(map[string]content.RepoNode)
	repo.builDirectory(repo.Node)
}
