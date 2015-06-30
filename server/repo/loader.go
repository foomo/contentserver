package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/foomo/contentserver/server/log"
	"github.com/foomo/contentserver/server/repo/content"
	"github.com/foomo/contentserver/server/responses"
	"github.com/foomo/contentserver/server/utils"
	//golog "log"
)

func (repo *Repo) updateRoutine() {
	go func() {
		for {
			log.Debug("update routine is about to select")
			select {
			case newDimension := <-repo.updateChannel:
				log.Debug("update routine received a new dimension: " + newDimension.Dimension)
				err := repo._updateDimension(newDimension.Dimension, newDimension.Node)
				log.Debug("update routine received result")
				if err != nil {
					log.Debug("	update routine error: " + err.Error())
				}
				repo.updateDoneChannel <- err
			}
		}
	}()
}

func (repo *Repo) updateDimension(dimension string, node *content.RepoNode) error {
	repo.updateChannel <- &RepoDimension{
		Dimension: dimension,
		Node:      node,
	}
	return <-repo.updateDoneChannel
}

// do not call directly, but only through channel
func (repo *Repo) _updateDimension(dimension string, newNode *content.RepoNode) error {
	newNode.WireParents()
	newDirectory := make(map[string]*content.RepoNode)
	newURIDirectory := make(map[string]*content.RepoNode)

	err := builDirectory(newNode, newDirectory, newURIDirectory)
	if err != nil {
		return errors.New("update dimension \"" + dimension + "\" failed when building its directory:: " + err.Error())
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
	if _, thereIsAnExistingURINode := uRIDirectory[dirNode.URI]; thereIsAnExistingURINode {
		return errors.New("duplicate uri: " + dirNode.URI + " (bad node id: " + dirNode.Id + ")")
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

func (repo *Repo) tryToRestoreCurrent() error {
	currentJSONBytes, err := repo.history.getCurrent()
	if err != nil {
		return err
	}
	return repo.loadJSONBytes(currentJSONBytes)
}

func (repo *Repo) update() (repoRuntime int64, jsonBytes []byte, err error) {
	startTimeRepo := time.Now().UnixNano()
	jsonBytes, err = utils.Get(repo.server)
	repoRuntime = time.Now().UnixNano() - startTimeRepo
	if err != nil {
		// we have no json to load - the repo server did not reply
		return repoRuntime, jsonBytes, err
	}
	nodes, err := loadNodesFromJSON(jsonBytes)
	if err != nil {
		// could not load nodes from json
		return repoRuntime, jsonBytes, err
	}
	err = repo.loadNodes(nodes)
	if err != nil {
		// repo failed to load nodes
		return repoRuntime, jsonBytes, err
	}
	return repoRuntime, jsonBytes, nil
}

func updateErrorHandler(err error, updateResponse *responses.Update) *responses.Update {
	log.Error(fmt.Sprintf("update error: %v", err))
	if updateResponse == nil {
		updateResponse = &responses.Update{}
	}
	updateResponse.Success = false
	updateResponse.ErrorMessage = fmt.Sprintf("%v", err)
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
	newDimensions := []string{}
	for dimension, newNode := range newNodes {
		newDimensions = append(newDimensions, dimension)
		log.Debug("loading nodes for dimension " + dimension)
		loadErr := repo.updateDimension(dimension, newNode)
		if loadErr != nil {
			log.Debug("	failed to load " + dimension + ": " + loadErr.Error())
			return loadErr
		}
	}
	dimensionIsValid := func(dimension string) bool {
		for _, newDimension := range newDimensions {
			if dimension == newDimension {
				return true
			}
		}
		return false
	}
	// we need to throw away orphaned dimensions
	for dimension := range repo.Directory {
		if !dimensionIsValid(dimension) {
			log.Notice("removing orphaned dimension:" + dimension)
			delete(repo.Directory, dimension)
		}
	}
	return nil
}
