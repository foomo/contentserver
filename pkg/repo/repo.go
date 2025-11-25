package repo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/foomo/contentserver/content"
	"github.com/foomo/contentserver/pkg/metrics"
	"github.com/foomo/contentserver/requests"
	"github.com/foomo/contentserver/responses"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const maxGetURIForNodeRecursionLevel = 1000

// Repo content repository
type (
	Repo struct {
		l                          *zap.Logger
		url                        string
		poll                       bool
		pollInterval               time.Duration
		pollVersion                string
		onLoaded                   func()
		loaded                     *atomic.Bool
		history                    *History
		httpClient                 *http.Client
		dimensionUpdateChannel     chan *RepoDimension
		dimensionUpdateDoneChannel chan error
		updateInProgressChannel    chan chan updateResponse
		directory                  map[string]*Dimension
		directoryLock              sync.RWMutex
		jsonBuffer                 *bytes.Buffer
		jsonBufferLock             sync.RWMutex
	}
	Option func(*Repo)
)

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

func New(l *zap.Logger, url string, history *History, opts ...Option) *Repo {
	inst := &Repo{
		l:                          l.Named("repo"),
		url:                        url,
		poll:                       false,
		loaded:                     &atomic.Bool{},
		pollInterval:               time.Minute,
		history:                    history,
		httpClient:                 http.DefaultClient,
		directory:                  map[string]*Dimension{},
		dimensionUpdateChannel:     make(chan *RepoDimension),
		dimensionUpdateDoneChannel: make(chan error),
		updateInProgressChannel:    make(chan chan updateResponse),
	}

	for _, opt := range opts {
		opt(inst)
	}

	return inst
}

// ------------------------------------------------------------------------------------------------
// ~ Options
// ------------------------------------------------------------------------------------------------

func WithHTTPClient(v *http.Client) Option {
	return func(o *Repo) {
		o.httpClient = v
	}
}

func WithPoll(v bool) Option {
	return func(o *Repo) {
		o.poll = v
	}
}

func WithPollInterval(v time.Duration) Option {
	return func(o *Repo) {
		o.pollInterval = v
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Getter
// ------------------------------------------------------------------------------------------------

func (r *Repo) Loaded() bool {
	return r.loaded.Load()
}

func (r *Repo) Directory() map[string]*Dimension {
	r.directoryLock.RLock()
	defer r.directoryLock.RUnlock()
	return r.directory
}

func (r *Repo) SetDirectory(v map[string]*Dimension) {
	r.directoryLock.Lock()
	defer r.directoryLock.Unlock()
	r.directory = v
}

func (r *Repo) JSONBufferBytes() []byte {
	r.jsonBufferLock.RLock()
	defer r.jsonBufferLock.RUnlock()
	return r.jsonBuffer.Bytes()
}

func (r *Repo) SetJSONBuffer(v *bytes.Buffer) {
	r.jsonBufferLock.Lock()
	defer r.jsonBufferLock.Unlock()
	r.jsonBuffer = v
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

func (r *Repo) OnLoaded(fn func()) {
	r.onLoaded = fn
}

// GetURIs get many uris at once
func (r *Repo) GetURIs(dimension string, ids []string) map[string]string {
	uris := map[string]string{}
	for _, id := range ids {
		uris[id] = r.getURI(dimension, id)
	}
	return uris
}

// GetNodes get nodes
func (r *Repo) GetNodes(nodes *requests.Nodes) map[string]*content.Node {
	return r.getNodes(nodes.Nodes, nodes.Env)
}

// GetContent resolves content and fetches nodes in one call. It combines those
// two tasks for performance reasons.
//
// In the first step it uses r.URI to look up content in all given
// r.Env.Dimensions of repo.Directory.
//
// In the second step it collects the requested nodes.
//
// those two steps are independent.
func (r *Repo) GetContent(req *requests.Content) (*content.SiteContent, error) {
	// add more input validation
	err := r.validateContentRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "repo.GetContent invalid request")
	}
	r.l.Debug("repo.GetContent", zap.String("URI", req.URI))
	c := content.NewSiteContent()
	resolved, resolvedURI, resolvedDimension, node := r.resolveContent(req.Env.Dimensions, req.URI)
	if resolved {
		if !node.CanBeAccessedByGroups(req.Env.Groups) {
			r.l.Warn("Resolved content cannot be accessed by specified group", zap.String("uri", req.URI))
			c.Status = content.StatusForbidden
		} else {
			r.l.Info("Content resolved", zap.String("uri", req.URI))
			c.Status = content.StatusOk
			c.Data = node.Data
		}
		c.MimeType = node.MimeType
		c.Dimension = resolvedDimension
		c.URI = resolvedURI
		c.Item = node.ToItem(req.DataFields)
		c.Path = node.GetPath(req.PathDataFields)
		// fetch URIs for all dimensions
		uris := make(map[string]string)
		for dimensionName := range r.Directory() {
			uris[dimensionName] = r.getURI(dimensionName, node.ID)
		}
		c.URIs = uris
	} else {
		r.l.Info("Content not found", zap.String("URI", req.URI))
		c.Status = content.StatusNotFound
		c.Dimension = req.Env.Dimensions[0]

		r.l.Debug("Failed to resolve, falling back to default dimension",
			zap.String("uri", req.URI),
			zap.String("default_dimension", req.Env.Dimensions[0]),
		)
		// r.Env.Dimensions is validated => we can access it
		resolvedDimension = req.Env.Dimensions[0]
	}

	// add navigation trees
	for _, node := range req.Nodes {
		if node.Dimension == "" {
			node.Dimension = resolvedDimension
		}
	}
	c.Nodes = r.getNodes(req.Nodes, req.Env)
	return c, nil
}

// GetRepo get the whole repo in all dimensions
func (r *Repo) GetRepo() map[string]*content.RepoNode {
	response := make(map[string]*content.RepoNode)
	for dimensionName, dimension := range r.Directory() {
		response[dimensionName] = dimension.Node
	}
	return response
}

// WriteRepoBytes writes the whole repo in all dimensions to the provided writer.
// It serves from the in-memory buffer, falling back to storage only when empty.
// The result is wrapped as service response, e.g: {"reply": <contentData>}
func (r *Repo) WriteRepoBytes(ctx context.Context, w io.Writer) error {
	r.jsonBufferLock.RLock()
	var data []byte
	if r.jsonBuffer != nil {
		data = r.jsonBuffer.Bytes()
	}
	r.jsonBufferLock.RUnlock()

	if len(data) == 0 {
		// Fallback to storage (cold start or not yet loaded)
		var buf bytes.Buffer
		if err := r.history.GetCurrent(ctx, &buf); err != nil {
			return fmt.Errorf("failed to read repo from storage: %w", err)
		}
		data = buf.Bytes()
	}

	if _, err := w.Write([]byte(`{"reply":`)); err != nil {
		return fmt.Errorf("failed to write repo JSON prefix: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write repo JSON data: %w", err)
	}
	if _, err := w.Write([]byte(`}`)); err != nil {
		return fmt.Errorf("failed to write repo JSON suffix: %w", err)
	}
	return nil
}

func (r *Repo) Update() (updateResponse *responses.Update) {
	floatSeconds := func(nanoSeconds int64) float64 {
		return float64(nanoSeconds) / float64(1000000000)
	}

	r.l.Info("Update triggered")
	// Log.Info(ansi.Yellow + "BUFFER LENGTH BEFORE tryUpdate(): " + strconv.Itoa(len(repo.jsonBuf.Bytes())) + ansi.Reset)

	start := time.Now()
	updateRepotime, err := r.tryUpdate()
	updateResponse = &responses.Update{}
	updateResponse.Stats.RepoRuntime = floatSeconds(updateRepotime)

	if err != nil {
		updateResponse.Success = false
		updateResponse.Stats.NumberOfNodes = -1
		updateResponse.Stats.NumberOfURIs = -1

		// let us try to restore the world from a file
		// Log.Info(ansi.Yellow + "BUFFER LENGTH AFTER ERROR: " + strconv.Itoa(len(r.jsonBuf.Bytes())) + ansi.Reset)
		// only try to restore if the update failed during processing

		if !errors.Is(err, ErrUpdateRejected) {
			updateResponse.ErrorMessage = err.Error()
			r.l.Error("Failed to update repository", zap.Error(err))

			restoreErr := r.tryToRestoreCurrent()
			if restoreErr != nil {
				r.l.Error("Failed to restore preceding repository version", zap.Error(restoreErr))
			} else {
				r.l.Info("Successfully restored current repository from local history")
			}
		}
	} else {
		updateResponse.Success = true
		// persist the currently loaded one
		historyErr := r.history.Add(context.Background(), r.JSONBufferBytes())
		if historyErr != nil {
			r.l.Error("Could not persist current repo in history", zap.Error(historyErr))
			metrics.HistoryPersistFailedCounter.WithLabelValues().Inc()
		} else {
			r.l.Info("Successfully persisted current repo to history")
		}
		// add some stats
		for _, dimension := range r.Directory() {
			updateResponse.Stats.NumberOfNodes += len(dimension.Directory)
			updateResponse.Stats.NumberOfURIs += len(dimension.URIDirectory)
		}
	}
	updateResponse.Stats.OwnRuntime = floatSeconds(time.Since(start).Nanoseconds()) - updateResponse.Stats.RepoRuntime
	return updateResponse
}

func (r *Repo) Start(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	l := r.l.Named("start")

	up := make(chan bool, 1)
	g.Go(func() error {
		l.Debug("starting update routine")
		up <- true
		return r.UpdateRoutine(gCtx)
	})
	l.Debug("waiting for UpdateRoutine")
	<-up

	g.Go(func() error {
		l.Debug("starting dimension update routine")
		up <- true
		return r.DimensionUpdateRoutine(gCtx)
	})
	l.Debug("waiting for DimensionUpdateRoutine")
	<-up

	l.Debug("trying to restore previous repo")
	if err := r.tryToRestoreCurrent(); errors.Is(err, os.ErrNotExist) {
		l.Info("previous repo content file does not exist")
	} else if err != nil {
		l.Warn("could not restore previous repo content", zap.Error(err))
	} else {
		l.Info("restored previous repo")
	}

	if r.poll {
		g.Go(func() error {
			l.Debug("starting poll routine")
			return r.PollRoutine(gCtx)
		})
	}

	if !r.Loaded() {
		l.Debug("trying to update initial state")
		if resp := r.Update(); !resp.Success {
			l.Error("failed to update initial state",
				zap.String("error", resp.ErrorMessage),
				zap.Int("num_modes", resp.Stats.NumberOfNodes),
				zap.Int("num_uris", resp.Stats.NumberOfURIs),
				zap.Float64("own_runtime", resp.Stats.OwnRuntime),
				zap.Float64("repo_runtime", resp.Stats.RepoRuntime),
			)
		}
	}

	return g.Wait()
}

// ------------------------------------------------------------------------------------------------
// ~ Private methods
// ------------------------------------------------------------------------------------------------

func (r *Repo) getNodes(nodeRequests map[string]*requests.Node, env *requests.Env) map[string]*content.Node {
	var (
		path  []*content.Item
		nodes = map[string]*content.Node{}
	)
	for nodeName, nodeRequest := range nodeRequests {
		if nodeName == "" || nodeRequest.ID == "" {
			r.l.Warn("invalid node request", zap.Error(errors.New("nodeName or nodeRequest.ID empty")))
			continue
		}
		r.l.Debug("adding node", zap.String("name", nodeName), zap.String("requestID", nodeRequest.ID))

		groups := env.Groups
		if len(nodeRequest.Groups) > 0 {
			groups = nodeRequest.Groups
		}

		dimensionNode, ok := r.Directory()[nodeRequest.Dimension]
		nodes[nodeName] = nil

		if !ok && nodeRequest.Dimension == "" {
			r.l.Debug("Could not get dimension root node", zap.String("dimension", nodeRequest.Dimension))
			for _, dimension := range env.Dimensions {
				dimensionNode, ok = r.Directory()[dimension]
				if ok {
					r.l.Debug("Found root node in env.Dimensions", zap.String("dimension", dimension))
					break
				}
				r.l.Debug("Could NOT find root node in env.Dimensions", zap.String("dimension", dimension))
			}
		}

		if !ok {
			r.l.Error("could not get dimension root node", zap.String("nodeRequest.Dimension", nodeRequest.Dimension))
			continue
		}

		treeNode, ok := dimensionNode.Directory[nodeRequest.ID]
		if !ok {
			r.l.Error("Invalid tree node requested",
				zap.String("nodeName", nodeName),
				zap.String("nodeID", nodeRequest.ID),
			)
			metrics.InvalidNodeTreeRequests.WithLabelValues().Inc()
			continue
		}
		nodes[nodeName] = r.getNode(treeNode, nodeRequest.Expand, nodeRequest.MimeTypes, path, 0, groups, nodeRequest.DataFields, nodeRequest.ExposeHiddenNodes)
	}
	return nodes
}

// resolveContent find content in a repository
func (r *Repo) resolveContent(dimensions []string, uri string) (resolved bool, resolvedURI string, resolvedDimension string, repoNode *content.RepoNode) {
	parts := strings.Split(uri, content.PathSeparator)
	r.l.Debug("repo.ResolveContent", zap.String("URI", uri))
	for i := len(parts); i > 0; i-- {
		testURI := strings.Join(parts[0:i], content.PathSeparator)
		if testURI == "" {
			testURI = content.PathSeparator
		}
		for _, dimension := range dimensions {
			if d, ok := r.Directory()[dimension]; ok {
				r.l.Debug("Checking node",
					zap.String("dimension", dimension),
					zap.String("URI", testURI),
				)
				if repoNode, ok := d.URIDirectory[testURI]; ok {
					resolved = true
					r.l.Debug("Node found", zap.String("URI", testURI), zap.String("destination", repoNode.DestinationID))
					if len(repoNode.DestinationID) > 0 {
						if destionationNode, destinationNodeOk := d.Directory[repoNode.DestinationID]; destinationNodeOk {
							repoNode = destionationNode
						}
					}
					return resolved, testURI, dimension, repoNode
				}
			}
		}
	}
	return
}

func (r *Repo) getURIForNode(dimension string, repoNode *content.RepoNode, recursionLevel int64) (uri string) {
	if len(repoNode.LinkID) == 0 {
		uri = repoNode.URI
		return
	}
	linkedNode, ok := r.Directory()[dimension].Directory[repoNode.LinkID]
	if ok {
		if recursionLevel > maxGetURIForNodeRecursionLevel {
			r.l.Error("maxGetURIForNodeRecursionLevel reached", zap.String("repoNode.ID", repoNode.ID), zap.String("linkID", repoNode.LinkID), zap.String("dimension", dimension))
			return ""
		}
		return r.getURIForNode(dimension, linkedNode, recursionLevel+1)
	}
	return
}

func (r *Repo) getURI(dimension string, id string) string {
	directory, ok := r.Directory()[dimension]
	if !ok {
		return ""
	}
	repoNode, ok := directory.Directory[id]
	if !ok {
		return ""
	}
	return r.getURIForNode(dimension, repoNode, 0)
}

func (r *Repo) getNode(
	repoNode *content.RepoNode,
	expanded bool,
	mimeTypes []string,
	path []*content.Item,
	level int,
	groups []string,
	dataFields []string,
	exposeHiddenNodes bool,
) *content.Node {
	node := content.NewNode()
	node.Item = repoNode.ToItem(dataFields)
	r.l.Debug("getNode", zap.String("ID", repoNode.ID))
	for _, childID := range repoNode.Index {
		childNode := repoNode.Nodes[childID]
		if (level == 0 || expanded || !expanded && childNode.InPath(path)) && (!childNode.Hidden || exposeHiddenNodes) && childNode.CanBeAccessedByGroups(groups) && childNode.IsOneOfTheseMimeTypes(mimeTypes) {
			node.Nodes[childID] = r.getNode(childNode, expanded, mimeTypes, path, level+1, groups, dataFields, exposeHiddenNodes)
			node.Index = append(node.Index, childID)
		}
	}
	return node
}

func (r *Repo) validateContentRequest(req *requests.Content) (err error) {
	if req == nil {
		return errors.New("request must not be nil")
	}
	if len(req.URI) == 0 {
		return errors.New("request URI must not be empty")
	}
	if req.Env == nil {
		return errors.New("request.Env must not be nil")
	}
	if len(req.Env.Dimensions) == 0 {
		return errors.New("request.Env.Dimensions must not be empty")
	}
	for _, envDimension := range req.Env.Dimensions {
		if !r.hasDimension(envDimension) {
			availableDimensions := make([]string, 0, len(r.Directory()))
			for availableDimension := range r.Directory() {
				availableDimensions = append(availableDimensions, availableDimension)
			}
			return errors.New(fmt.Sprint(
				"unknown dimension ", envDimension,
				" in r.Env must be one of ", availableDimensions,
				" repo has ", len(availableDimensions), " dimensions",
			))
		}
	}
	return nil
}

func (r *Repo) hasDimension(d string) bool {
	_, hasDimension := r.Directory()[d]
	return hasDimension
}
