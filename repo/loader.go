package repo

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mgutz/ansi"

	"github.com/foomo/contentserver/content"
	. "github.com/foomo/contentserver/logger"
	"github.com/foomo/contentserver/status"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var (
	json              = jsoniter.ConfigCompatibleWithStandardLibrary
	errUpdateRejected = errors.New("update rejected: queue full")
)

type updateResponse struct {
	repoRuntime int64
	err         error
}

func (repo *Repo) updateRoutine() {
	for {
		select {
		case resChan := <-repo.updateInProgressChannel:
			Log.Info("waiting for update to complete", zap.String("chan", fmt.Sprintf("%p", resChan)))
			start := time.Now()

			repoRuntime, errUpdate := repo.update()
			if errUpdate != nil {
				status.M.UpdatesFailedCounter.WithLabelValues(errUpdate.Error()).Inc()
			}

			resChan <- updateResponse{
				repoRuntime: repoRuntime,
				err:         errUpdate,
			}

			duration := time.Since(start)
			Log.Info("update completed", zap.Duration("duration", duration), zap.String("chan", fmt.Sprintf("%p", resChan)))
			status.M.UpdatesCompletedCounter.WithLabelValues().Inc()
			status.M.UpdateDuration.WithLabelValues().Observe(duration.Seconds())
		}
	}
}

func (repo *Repo) dimensionUpdateRoutine() {
	for newDimension := range repo.dimensionUpdateChannel {
		Log.Info("update routine received a new dimension", zap.String("dimension", newDimension.Dimension))

		err := repo._updateDimension(newDimension.Dimension, newDimension.Node)
		Log.Info("update routine received result")
		if err != nil {
			Log.Debug("update dimension failed", zap.Error(err))
		}
		repo.dimensionUpdateDoneChannel <- err
	}
}

func (repo *Repo) updateDimension(dimension string, node *content.RepoNode) error {
	Log.Debug("trying to push dimension into update channel", zap.String("dimension", dimension), zap.String("nodeName", node.Name))
	repo.dimensionUpdateChannel <- &repoDimension{
		Dimension: dimension,
		Node:      node,
	}
	Log.Debug("waiting for done signal")
	return <-repo.dimensionUpdateDoneChannel
}

// do not call directly, but only through channel
func (repo *Repo) _updateDimension(dimension string, newNode *content.RepoNode) error {
	newNode.WireParents()

	var (
		newDirectory    = make(map[string]*content.RepoNode)
		newURIDirectory = make(map[string]*content.RepoNode)
		err             = buildDirectory(newNode, newDirectory, newURIDirectory)
	)
	if err != nil {
		return errors.New("update dimension \"" + dimension + "\" failed when building its directory:: " + err.Error())
	}
	err = wireAliases(newDirectory)
	if err != nil {
		return err
	}

	// ---------------------------------------------

	// copy old datastructure to prevent concurrent map access
	// collect other dimension in the Directory
	newRepoDirectory := map[string]*Dimension{}
	for d, D := range repo.Directory {
		if d != dimension {
			newRepoDirectory[d] = D
		}
	}

	// add the new dimension
	newRepoDirectory[dimension] = &Dimension{
		Node:         newNode,
		Directory:    newDirectory,
		URIDirectory: newURIDirectory,
	}
	repo.Directory = newRepoDirectory

	// ---------------------------------------------

	// @TODO: why not update only the dimension that has changed instead?
	// repo.Directory[dimension] = &Dimension{
	// 	Node:         newNode,
	// 	Directory:    newDirectory,
	// 	URIDirectory: newURIDirectory,
	// }

	// ---------------------------------------------

	return nil
}

func buildDirectory(dirNode *content.RepoNode, directory map[string]*content.RepoNode, uRIDirectory map[string]*content.RepoNode) error {

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
		err := buildDirectory(childNode, directory, uRIDirectory)
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

func (repo *Repo) loadNodesFromJSON() (nodes map[string]*content.RepoNode, err error) {
	nodes = make(map[string]*content.RepoNode)
	err = json.Unmarshal(repo.jsonBuf.Bytes(), &nodes)
	return nodes, err
}

func (repo *Repo) tryToRestoreCurrent() (err error) {
	err = repo.history.getCurrent(&repo.jsonBuf)
	if err != nil {
		return err
	}
	return repo.loadJSONBytes()
}

func (repo *Repo) get(URL string) (err error) {
	response, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Bad HTTP Response: %q", response.Status)
	}

	Log.Info(ansi.Red + "RESETTING BUFFER" + ansi.Reset)
	repo.jsonBuf.Reset()

	Log.Info(ansi.Green + "LOADING DATA INTO BUFFER" + ansi.Reset)
	_, err = io.Copy(&repo.jsonBuf, response.Body)
	return err
}

func (repo *Repo) update() (repoRuntime int64, err error) {
	startTimeRepo := time.Now().UnixNano()
	err = repo.get(repo.server)
	repoRuntime = time.Now().UnixNano() - startTimeRepo
	if err != nil {
		// we have no json to load - the repo server did not reply
		Log.Debug("failed to load json", zap.Error(err))
		return repoRuntime, err
	}
	Log.Debug("loading json", zap.String("server", repo.server), zap.Int("length", len(repo.jsonBuf.Bytes())))
	nodes, err := repo.loadNodesFromJSON()
	if err != nil {
		// could not load nodes from json
		return repoRuntime, err
	}
	err = repo.loadNodes(nodes)
	if err != nil {
		// repo failed to load nodes
		return repoRuntime, err
	}
	return repoRuntime, nil
}

// limit ressources and allow only one update request at once
func (repo *Repo) tryUpdate() (repoRuntime int64, err error) {
	c := make(chan updateResponse)
	select {
	case repo.updateInProgressChannel <- c:
		Log.Info("update request added to queue")
		ur := <-c
		return ur.repoRuntime, ur.err
	default:
		Log.Info("update request rejected, queue is full")
		status.M.UpdatesRejectedCounter.WithLabelValues().Inc()
		return 0, errUpdateRejected
	}
}

func (repo *Repo) loadJSONBytes() error {
	nodes, err := repo.loadNodesFromJSON()
	if err != nil {
		data := repo.jsonBuf.Bytes()

		if len(data) > 10 {
			Log.Debug("could not parse json",
				zap.String("jsonStart", string(data[:10])),
				zap.String("jsonStart", string(data[len(data)-10:])),
			)
		}
		return err
	}

	err = repo.loadNodes(nodes)
	if err == nil {
		historyErr := repo.history.add(repo.jsonBuf.Bytes())
		if historyErr != nil {
			Log.Error("could not add valid json to history", zap.Error(historyErr))
			status.M.HistoryPersistFailedCounter.WithLabelValues(historyErr.Error()).Inc()
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
