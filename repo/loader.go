package repo

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/logger"
	"github.com/foomo/contentserver/status"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
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
	for resChan := range repo.updateInProgressChannel {
		log := logger.Log.With(zap.String("updateRunID", uuid.New().String()))

		log.Info("Content update started")
		start := time.Now()

		repoRuntime, err := repo.update(context.Background())
		if err != nil {
			log.Error("Content update failed", zap.Error(err))
			status.M.UpdatesFailedCounter.WithLabelValues().Inc()
		} else {
			log.Info("Content update success")
			status.M.UpdatesCompletedCounter.WithLabelValues().Inc()
		}

		resChan <- updateResponse{
			repoRuntime: repoRuntime,
			err:         err,
		}

		status.M.UpdateDuration.WithLabelValues().Observe(time.Since(start).Seconds())
	}
}

func (repo *Repo) dimensionUpdateRoutine() {
	for newDimension := range repo.dimensionUpdateChannel {
		logger.Log.Info("dimensionUpdateRoutine received a new dimension", zap.String("dimension", newDimension.Dimension))

		err := repo._updateDimension(newDimension.Dimension, newDimension.Node)
		logger.Log.Info("dimensionUpdateRoutine received result")
		if err != nil {
			logger.Log.Debug("update dimension failed", zap.Error(err))
		}
		repo.dimensionUpdateDoneChannel <- err
	}
}

func (repo *Repo) updateDimension(dimension string, node *content.RepoNode) error {
	logger.Log.Debug("trying to push dimension into update channel", zap.String("dimension", dimension), zap.String("nodeName", node.Name))
	repo.dimensionUpdateChannel <- &repoDimension{
		Dimension: dimension,
		Node:      node,
	}
	logger.Log.Debug("waiting for done signal")
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
	if err != nil {
		logger.Log.Error("Failed to deserialize nodes", zap.Error(err))
		return nil, errors.New("failed to deserialize nodes")
	}
	return nodes, nil
}

func (repo *Repo) tryToRestoreCurrent() (err error) {
	err = repo.history.getCurrent(&repo.jsonBuf)
	if err != nil {
		return err
	}
	return repo.loadJSONBytes()
}

func (repo *Repo) get(URL string) error {
	response, err := repo.httpClient.Get(URL)
	if err != nil {
		return errors.Wrap(err, "failed to get repo")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf("bad response code from repository %q want %q", response.Status, http.StatusOK)
	}

	// Log.Info(ansi.Red + "RESETTING BUFFER" + ansi.Reset)
	repo.jsonBuf.Reset()

	// Log.Info(ansi.Green + "LOADING DATA INTO BUFFER" + ansi.Reset)
	_, err = io.Copy(&repo.jsonBuf, response.Body)
	if err != nil {
		return errors.Wrap(err, "failed to copy IO stream")
	}

	return nil
}

func (repo *Repo) update(ctx context.Context) (repoRuntime int64, err error) {
	startTimeRepo := time.Now().UnixNano()

	repoURL := repo.server
	if repo.pollForUpdates {
		resp, err := repo.httpClient.Get(repo.server)
		if err != nil {
			return repoRuntime, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return repoRuntime, errors.New("could not poll latest repo download url - non 200 response")
		}
		responseBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return repoRuntime, errors.New("could not poll latest repo download url, could not read body")
		}
		repoURL = string(responseBytes)
		if repoURL == repo.pollVersion {
			logger.Log.Info(
				"repo is up to date",
				zap.String("pollVersion", repo.pollVersion),
			)
			// already up to date
			return repoRuntime, nil
		} else {
			logger.Log.Info(
				"new repo poll version",
				zap.String("pollVersion", repo.pollVersion),
			)
		}
	}

	err = repo.get(repoURL)
	repoRuntime = time.Now().UnixNano() - startTimeRepo
	if err != nil {
		// we have no json to load - the repo server did not reply
		logger.Log.Debug("Failed to load json", zap.Error(err))
		return repoRuntime, err
	}
	logger.Log.Debug("loading json", zap.String("server", repoURL), zap.Int("length", len(repo.jsonBuf.Bytes())))
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
	if repo.pollForUpdates {
		repo.pollVersion = repoURL
	}
	return repoRuntime, nil
}

// limit ressources and allow only one update request at once
func (repo *Repo) tryUpdate() (repoRuntime int64, err error) {
	c := make(chan updateResponse)
	select {
	case repo.updateInProgressChannel <- c:
		logger.Log.Info("update request added to queue")
		ur := <-c
		return ur.repoRuntime, ur.err
	default:
		logger.Log.Info("update request accepted, will be processed after the previous update")
		return 0, errUpdateRejected
	}
}

func (repo *Repo) loadJSONBytes() error {
	nodes, err := repo.loadNodesFromJSON()
	if err != nil {
		data := repo.jsonBuf.Bytes()

		if len(data) > 10 {
			logger.Log.Debug("could not parse json",
				zap.String("jsonStart", string(data[:10])),
				zap.String("jsonStart", string(data[len(data)-10:])),
			)
		}
		return err
	}

	err = repo.loadNodes(nodes)
	if err == nil {
		errHistory := repo.history.add(repo.jsonBuf.Bytes())
		if errHistory != nil {
			logger.Log.Error("Could not add valid JSON to history", zap.Error(errHistory))
			status.M.HistoryPersistFailedCounter.WithLabelValues().Inc()
		} else {
			logger.Log.Info("Added valid JSON to history")
		}
	}
	return err
}

func (repo *Repo) loadNodes(newNodes map[string]*content.RepoNode) error {
	var newDimensions []string
	var err error
	for dimension, newNode := range newNodes {
		newDimensions = append(newDimensions, dimension)
		logger.Log.Debug("Loading nodes for dimension", zap.String("dimension", dimension))
		errLoad := repo.updateDimension(dimension, newNode)
		if errLoad != nil {
			err = multierr.Append(err, errLoad)
		}
	}
	if err != nil {
		return errors.Wrap(err, "failed to update dimension")
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
			logger.Log.Info("Removing orphaned dimension", zap.String("dimension", dimension))
			delete(repo.Directory, dimension)
		}
	}
	return nil
}
