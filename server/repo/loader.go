package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/foomo/contentserver/server/log"
	"github.com/foomo/contentserver/server/repo/content"
	"github.com/foomo/contentserver/server/responses"
	"github.com/foomo/contentserver/server/utils"
	"time"
	//golog "log"
)

func (repo *Repo) updateRoutine() {
	repo.updateChannel = make(chan *RepoDimension)
	repo.updateDoneChannel = make(chan error)
	go func() {
		for {
			select {
			case newDimension := <-repo.updateChannel:
				repo.updateDoneChannel <- repo.updateDimension(newDimension.Dimension, newDimension.Node)
			}
		}
	}()
}

func (repo *Repo) UpdateDimension(dimension string, node *content.RepoNode) error {
	repo.updateChannel <- &RepoDimension{
		Dimension: dimension,
		Node:      node,
	}
	return <-repo.updateDoneChannel
}

func (repo *Repo) updateDimension(dimension string, newNode *content.RepoNode) error {
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

func loadNodesFromJSON(jsonBytes []byte) (nodes map[string]*content.RepoNode, err error) {
	nodes = make(map[string]*content.RepoNode)
	err = json.Unmarshal(jsonBytes, &nodes)
	return nodes, err
}

func (repo *Repo) Update() (updateResponse *responses.Update) {
	floatSeconds := func(nanoSeconds int64) float64 {
		return float64(nanoSeconds / 1000000)
	}
	startTime := time.Now().UnixNano()
	updateRepotime, updateErr := repo.update()
	updateResponse = responses.NewUpdate()
	updateResponse.Stats.RepoRuntime = floatSeconds(updateRepotime)
	if updateErr != nil {
		// let us try to restore the world from a file
		log.Error("could not update repository:" + updateErr.Error())
		restoreErr := repo.tryToRestoreCurrent()
		if restoreErr == nil {
			log.Error("failed to restore preceding repo version")
		} else {
			log.Record("restored current repo from local history")
		}
	} else {
		// add some stats
		for dimension, _ := range repo.Directory {
			updateResponse.Stats.NumberOfNodes += len(repo.Directory[dimension].Directory)
			updateResponse.Stats.NumberOfURIs += len(repo.Directory[dimension].URIDirectory)
		}

	}
	updateResponse.Stats.OwnRuntime = floatSeconds(startTime-time.Now().UnixNano()) - updateResponse.Stats.RepoRuntime
	return updateResponse
}

func (repo *Repo) tryToRestoreCurrent() error {
	currentJsonBytes, err := repo.history.getCurrent()
	if err != nil {
		return err
	}
	return repo.loadJSONBytes(currentJsonBytes)
}

func (repo *Repo) update() (repoRuntime int64, err error) {
	startTimeRepo := time.Now().UnixNano()
	jsonBytes, err := utils.Get(repo.server)
	repoRuntime = time.Now().UnixNano() - startTimeRepo
	if err != nil {
		// we have no json to load - the repo server did not reply
		return
	} else {
		nodes, err := loadNodesFromJSON(jsonBytes)
		if err != nil {
			// could not load nodes from json
			return repoRuntime, err
		}
		err = repo.loadNodes(nodes)
		if err != nil {
			// repo failed to load nodes
			return repoRuntime, err
		}
	}
	return repoRuntime, nil

	/*

		log.Debug("loaded nodes for dimension " + dimension)
		_, dimensionOk := repo.Directory[dimension]
		if dimensionOk {
			updateResponse.Stats.NumberOfNodes += len(repo.Directory[dimension].Directory)
			updateResponse.Stats.NumberOfURIs += len(repo.Directory[dimension].URIDirectory)
		}



		jsonBytes
			newNodes := loadNodesFromJSON(jsonBytes)

		data, err := utils.GetRepo(repo.server, newNodes)
		updateResponse.Stats.RepoRuntime =
		startTimeOwn := time.Now()
		if err == nil {
		}
		updateResponse.Success = (err != nil)

		doneHandler := func() *responses.Update {
			updateResponse.Stats.OwnRuntime = time.Now().Sub(startTimeOwn).Seconds()
			return updateResponse
		}

		if updateResponse.Success {
			log.Debug("going to load dimensions from" + utils.ToJSON(newNodes))
	*/
}

func updateErrorHandler(err error, updateResponse *responses.Update) *responses.Update {
	log.Error(fmt.Sprintf("update error: %", err))
	if updateResponse == nil {
		updateResponse = responses.NewUpdate()
	}
	updateResponse.Success = false
	updateResponse.ErrorMessage = fmt.Sprintf("%", err)
	updateResponse.Stats.NumberOfNodes = -1
	updateResponse.Stats.NumberOfURIs = -1
	return updateResponse
}

func (repo *Repo) loadJSONBytes(jsonBytes []byte) error {
	nodes, err := loadNodesFromJSON(jsonBytes)
	if err != nil {
		return err
	}
	err = repo.loadNodes(nodes)
	if err == nil {
		historyErr := repo.history.add(jsonBytes)
		if historyErr != nil {
			log.Warning("could not add valid json to history:" + historyErr.Error())
		} else {
			log.Record("added valid json to history")
		}
	}
	return err
}

func (repo *Repo) loadNodes(newNodes map[string]*content.RepoNode) error {
	for dimension, newNode := range newNodes {
		log.Debug("loading nodes for dimension " + dimension)
		loadErr := repo.UpdateDimension(dimension, newNode)
		if loadErr != nil {
			log.Debug("	failed to load " + dimension + ": " + loadErr.Error())
			return loadErr
		}
	}
	// we need to throw away orphaned nodes
	return nil
}
