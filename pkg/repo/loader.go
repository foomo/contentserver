package repo

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/pkg/metrics"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

var (
	json              = jsoniter.ConfigCompatibleWithStandardLibrary
	ErrUpdateRejected = errors.New("update rejected: queue full")
)

type updateResponse struct {
	repoRuntime int64
	err         error
}

func (r *Repo) PollRoutine(ctx context.Context) error {
	l := r.l.Named("routine.poll")
	ticker := time.NewTicker(r.pollInterval)
	for {
		select {
		case <-ctx.Done():
			l.Debug("routine canceled", zap.Error(ctx.Err()))
			return nil
		case <-ticker.C:
			chanReponse := make(chan updateResponse)
			r.updateInProgressChannel <- chanReponse
			response := <-chanReponse
			if response.err == nil {
				l.Info("update success", zap.String("revision", r.pollVersion))
			} else {
				l.Error("update failed", zap.Error(response.err))
			}
		}
	}
}

func (r *Repo) UpdateRoutine(ctx context.Context) error {
	l := r.l.Named("routine.update")
	for {
		select {
		case <-ctx.Done():
			l.Debug("routine canceled", zap.Error(ctx.Err()))
			return nil
		case resChan := <-r.updateInProgressChannel:
			start := time.Now()
			l := l.With(zap.String("run_id", uuid.New().String()))

			l.Info("update started")

			repoRuntime, err := r.update(context.WithoutCancel(ctx))
			if err != nil {
				l.Error("update failed", zap.Error(err))
				metrics.UpdatesFailedCounter.WithLabelValues().Inc()
			} else {
				if !r.Loaded() {
					r.loaded.Store(true)
					l.Info("initial update success")
					if r.onLoaded != nil {
						r.onLoaded()
					}
				} else {
					l.Info("update success")
				}
				metrics.UpdatesCompletedCounter.WithLabelValues().Inc()
			}

			resChan <- updateResponse{
				repoRuntime: repoRuntime,
				err:         err,
			}

			metrics.UpdateDuration.WithLabelValues().Observe(time.Since(start).Seconds())
		}
	}
}

func (r *Repo) DimensionUpdateRoutine(ctx context.Context) error {
	l := r.l.Named("routine.dimensionUpdate")
	for {
		select {
		case <-ctx.Done():
			l.Debug("routine canceled",
				zap.Error(ctx.Err()),
			)
			return nil
		case newDimension := <-r.dimensionUpdateChannel:
			l.Debug("received a new dimension", zap.String("dimension", newDimension.Dimension))

			err := r._updateDimension(newDimension.Dimension, newDimension.Node)
			l.Info("received result")
			if err != nil {
				l.Debug("update failed", zap.Error(err))
			}
			r.dimensionUpdateDoneChannel <- err
		}
	}
}

func (r *Repo) updateDimension(dimension string, node *content.RepoNode) error {
	r.l.Debug("trying to push dimension into update channel", zap.String("dimension", dimension), zap.String("nodeName", node.Name))
	r.dimensionUpdateChannel <- &RepoDimension{
		Dimension: dimension,
		Node:      node,
	}
	r.l.Debug("waiting for done signal")
	return <-r.dimensionUpdateDoneChannel
}

// do not call directly, but only through channel
func (r *Repo) _updateDimension(dimension string, newNode *content.RepoNode) error {
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
	for d, D := range r.Directory() {
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
	r.SetDirectory(newRepoDirectory)

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
	existingNode, ok := directory[dirNode.ID]
	if ok {
		return errors.New("duplicate node with id:" + existingNode.ID)
	}
	directory[dirNode.ID] = dirNode
	// todo handle duplicate uris
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

func (r *Repo) loadNodesFromJSON() (nodes map[string]*content.RepoNode, err error) {
	nodes = make(map[string]*content.RepoNode)
	err = json.Unmarshal(r.JSONBufferBytes(), &nodes)
	if err != nil {
		r.l.Error("Failed to deserialize nodes", zap.Error(err))
		return nil, errors.New("failed to deserialize nodes")
	}
	return nodes, nil
}

func (r *Repo) tryToRestoreCurrent() error {
	buffer := &bytes.Buffer{}
	err := r.history.GetCurrent(buffer)
	if err != nil {
		return err
	}
	r.SetJSONBuffer(buffer)
	return r.loadJSONBytes()
}

func (r *Repo) get(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create get repo request")
	}
	response, err := r.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to get repo")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf("bad response code from repository %q want %q", response.Status, http.StatusOK)
	}

	// Log.Info(ansi.Red + "RESETTING BUFFER" + ansi.Reset)
	buffer := &bytes.Buffer{}

	// Log.Info(ansi.Green + "LOADING DATA INTO BUFFER" + ansi.Reset)
	_, err = io.Copy(buffer, response.Body)
	if err != nil {
		return errors.Wrap(err, "failed to copy IO stream")
	}
	r.SetJSONBuffer(buffer)

	return nil
}

func (r *Repo) update(ctx context.Context) (repoRuntime int64, err error) {
	startTimeRepo := time.Now().UnixNano()

	repoURL := r.url
	if r.poll {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.url, nil)
		if err != nil {
			return repoRuntime, err
		}
		resp, err := r.httpClient.Do(req)
		if err != nil {
			return repoRuntime, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return repoRuntime, errors.New("could not poll latest repo download url - non 200 response")
		}
		responseBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return repoRuntime, errors.New("could not poll latest repo download url, could not read body")
		}
		repoURL = string(responseBytes)
		if repoURL == r.pollVersion {
			r.l.Info(
				"repo is up to date",
				zap.String("pollVersion", r.pollVersion),
			)
			// already up to date
			return repoRuntime, nil
		}
		r.l.Info(
			"new repo poll version",
			zap.String("pollVersion", r.pollVersion),
		)
	}

	err = r.get(ctx, repoURL)
	repoRuntime = time.Now().UnixNano() - startTimeRepo
	if err != nil {
		// we have no json to load - the repo server did not reply
		r.l.Debug("failed to load json", zap.Error(err))
		return repoRuntime, err
	}
	r.l.Debug("loading json", zap.String("server", repoURL), zap.Int("length", len(r.JSONBufferBytes())))
	nodes, err := r.loadNodesFromJSON()
	if err != nil {
		// could not load nodes from json
		return repoRuntime, err
	}
	err = r.loadNodes(nodes)
	if err != nil {
		// repo failed to load nodes
		return repoRuntime, err
	}
	if r.poll {
		r.pollVersion = repoURL
	}
	return repoRuntime, nil
}

// limit ressources and allow only one update request at once
func (r *Repo) tryUpdate() (repoRuntime int64, err error) {
	c := make(chan updateResponse)
	select {
	case r.updateInProgressChannel <- c:
		r.l.Debug("update request added to queue")
		ur := <-c
		return ur.repoRuntime, ur.err
	default:
		r.l.Info("update request accepted, will be processed after the previous update")
		return 0, ErrUpdateRejected
	}
}

func (r *Repo) loadJSONBytes() error {
	nodes, err := r.loadNodesFromJSON()
	if err != nil {
		data := r.JSONBufferBytes()

		if len(data) > 10 {
			r.l.Debug("could not parse json",
				zap.String("jsonStart", string(data[:10])),
				zap.String("jsonStart", string(data[len(data)-10:])),
			)
		}
		return err
	}

	err = r.loadNodes(nodes)
	if err == nil {
		errHistory := r.history.Add(r.JSONBufferBytes())
		if errHistory != nil {
			r.l.Error("Could not add valid JSON to history", zap.Error(errHistory))
			metrics.HistoryPersistFailedCounter.WithLabelValues().Inc()
		} else {
			r.l.Info("added valid JSON to history")
		}
	}
	return err
}

func (r *Repo) loadNodes(newNodes map[string]*content.RepoNode) error {
	var err error
	newDimensions := make([]string, 0, len(newNodes))
	for dimension, newNode := range newNodes {
		newDimensions = append(newDimensions, dimension)
		r.l.Debug("loading nodes for dimension", zap.String("dimension", dimension))
		errLoad := r.updateDimension(dimension, newNode)
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
	directory := map[string]*Dimension{}
	for dimension, value := range r.Directory() {
		if !dimensionIsValid(dimension) {
			r.l.Info("removing orphaned dimension", zap.String("dimension", dimension))
			continue
		}
		directory[dimension] = value
	}
	r.SetDirectory(directory)
	return nil
}
