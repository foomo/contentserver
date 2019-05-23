package repo

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/foomo/contentserver/content"
	. "github.com/foomo/contentserver/logger"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type updateResponse struct {
	repoRuntime int64
	jsonBytes   []byte
	err         error
}

func (repo *Repo) updateRoutine() {
	for newDimension := range repo.updateChannel {
		Log.Info("update routine received a new dimension", zap.String("dimension", newDimension.Dimension))

		err := repo._updateDimension(newDimension.Dimension, newDimension.Node)
		Log.Info("update routine received result")
		if err != nil {
			Log.Debug("update dimension failed", zap.Error(err))
		}
		repo.updateDoneChannel <- err
	}
}

func (repo *Repo) updateDimension(dimension string, node *content.RepoNode) error {
	Log.Debug("trying to push dimension into update channel", zap.String("dimension", dimension), zap.String("nodeName", node.Name))
	repo.updateChannel <- &repoDimension{
		Dimension: dimension,
		Node:      node,
	}
	Log.Debug("waiting for done signal")
	return <-repo.updateDoneChannel
}

// do not call directly, but only through channel
func (repo *Repo) _updateDimension(dimension string, newNode *content.RepoNode) error {
	newNode.WireParents()

	var (
		newDirectory    = make(map[string]*content.RepoNode)
		newURIDirectory = make(map[string]*content.RepoNode)
		err             = builDirectory(newNode, newDirectory, newURIDirectory)
	)
	if err != nil {
		return errors.New("update dimension \"" + dimension + "\" failed when building its directory:: " + err.Error())
	}
	err = wireAliases(newDirectory)
	if err != nil {
		return err
	}

	newRepoDirectory := map[string]*Dimension{}
	for d, D := range repo.Directory {
		if d != dimension {
			newRepoDirectory[d] = D
		}
	}
	newRepoDirectory[dimension] = &Dimension{
		Node:         newNode,
		Directory:    newDirectory,
		URIDirectory: newURIDirectory,
	}
	repo.Directory = newRepoDirectory
	return nil
}

func builDirectory(dirNode *content.RepoNode, directory map[string]*content.RepoNode, uRIDirectory map[string]*content.RepoNode) error {

	// Log.Debug("buildDirectory", zap.String("ID", dirNode.ID))

	existingNode, ok := directory[dirNode.ID]
	if ok {
		return errors.New("duplicate node with id:" + existingNode.ID)
	}
	directory[dirNode.ID] = dirNode
	//todo handle duplicate uris
	if _, thereIsAnExistingURINode := uRIDirectory[dirNode.URI]; thereIsAnExistingURINode {
		return errors.New("duplicate uri: " + dirNode.URI + " (bad node id: " + dirNode.ID + ")")
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
		if len(repoNode.LinkID) > 0 {
			if destinationNode, ok := directory[repoNode.LinkID]; ok {
				repoNode.URI = destinationNode.URI
			} else {
				return errors.New("that link id points nowhere " + repoNode.LinkID + " from " + repoNode.ID)
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

func (repo *Repo) tryToRestoreCurrent() error {
	currentJSONBytes, err := repo.history.getCurrent()
	if err != nil {
		return err
	}
	return repo.loadJSONBytes(currentJSONBytes)
}

func get(URL string) (data []byte, err error) {
	response, err := http.Get(URL)
	if err != nil {
		return data, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return data, fmt.Errorf("Bad HTTP Response: %q", response.Status)
	}
	return ioutil.ReadAll(response.Body)
}

func (repo *Repo) update() (repoRuntime int64, jsonBytes []byte, err error) {
	startTimeRepo := time.Now().UnixNano()
	jsonBytes, err = get(repo.server)
	repoRuntime = time.Now().UnixNano() - startTimeRepo
	if err != nil {
		// we have no json to load - the repo server did not reply
		Log.Debug("failed to load json", zap.Error(err))
		return repoRuntime, jsonBytes, err
	}
	Log.Debug("loading json", zap.String("server", repo.server), zap.Int("length", len(jsonBytes)))
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

// limit ressources and allow only one update request at once
func (repo *Repo) tryUpdate() (repoRuntime int64, jsonBytes []byte, err error) {
	c := make(chan updateResponse)
	select {
	case repo.updateInProgressChannel <- c:
		Log.Info("update request added to queue")
		ur := <-c
		return ur.repoRuntime, ur.jsonBytes, ur.err
	default:
		Log.Info("update request ignored, queue is full")
		return 0, nil, errors.New("queue full")
	}
}

func (repo *Repo) loadJSONBytes(jsonBytes []byte) error {
	nodes, err := loadNodesFromJSON(jsonBytes)
	if err != nil {
		Log.Debug("could not parse json", zap.String("json", string(jsonBytes)))
		return err
	}
	err = repo.loadNodes(nodes)
	if err == nil {
		historyErr := repo.history.add(jsonBytes)
		if historyErr != nil {
			Log.Error("could not add valid json to history", zap.Error(historyErr))
		} else {
			Log.Info("added valid json to history")
		}
		cleanUpErr := repo.history.cleanup()
		if cleanUpErr != nil {
			Log.Error("an error occured while cleaning up my history", zap.Error(cleanUpErr))
		} else {
			Log.Info("cleaned up history")
		}
	}
	return err
}

func (repo *Repo) loadNodes(newNodes map[string]*content.RepoNode) error {
	newDimensions := []string{}
	for dimension, newNode := range newNodes {
		newDimensions = append(newDimensions, dimension)
		Log.Debug("loading nodes for dimension", zap.String("dimension", dimension))
		loadErr := repo.updateDimension(dimension, newNode)
		if loadErr != nil {
			Log.Debug("failed to load", zap.String("dimension", dimension), zap.Error(loadErr))
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
			Log.Info("removing orphaned dimension", zap.String("dimension", dimension))
			delete(repo.Directory, dimension)
		}
	}
	return nil
}
