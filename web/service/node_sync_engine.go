package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"x-ui/database"
	"x-ui/database/model"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNodeSyncNodeIDRequired     = errors.New("node ID is required")
	ErrNodeSyncItemIDRequired     = errors.New("node sync item ID is required")
	ErrNodeSyncDuplicateItemID    = errors.New("node sync item ID must be unique within a snapshot")
	ErrNodeSyncSnapshotIncomplete = errors.New("node sync snapshot is incomplete")
	ErrNodeSyncNodeNotFound       = errors.New("configured node was not found")
	ErrNodeSyncNodeDisabled       = errors.New("configured node is disabled")
	ErrNodeSyncNodeURLInvalid     = errors.New("configured node URL is invalid")
	ErrNodeSyncTokenInvalid       = errors.New("node sync token is invalid")
	ErrNodeSyncRemoteUnavailable  = errors.New("remote node is unavailable")
	ErrNodeSyncRemoteRejected     = errors.New("remote node rejected the request")
	ErrNodeSyncRemoteMalformed    = errors.New("remote node response is invalid")
	ErrNodeSyncRemoteIncomplete   = errors.New("remote node snapshot is incomplete")
)

const (
	nodeSyncRequestTimeout   = 5 * time.Second
	nodeSyncMaxResponseBytes = 1 << 20
)

// NodeSyncItem is the comparison-safe portion of an item snapshot.
// Revision is used only to decide whether an item needs an update.
type NodeSyncItem struct {
	ID       string `json:"id"`
	Revision string `json:"revision"`
}

// NodeSyncSnapshot is one side of a node synchronization comparison.
type NodeSyncSnapshot struct {
	Complete bool           `json:"complete"`
	Items    []NodeSyncItem `json:"items"`
}

// NodeSyncPlanInput contains the snapshots used to calculate a synchronization plan.
type NodeSyncPlanInput struct {
	NodeID int              `json:"nodeId"`
	Local  NodeSyncSnapshot `json:"local"`
	Remote NodeSyncSnapshot `json:"remote"`
	DryRun bool             `json:"dryRun"`
}

// NodeSyncAction describes one planned snapshot delta; it never executes a remote operation.
type NodeSyncAction string

const (
	NodeSyncAdd    NodeSyncAction = "add"
	NodeSyncUpdate NodeSyncAction = "update"
	NodeSyncRemove NodeSyncAction = "remove"
)

// NodeSyncOperation identifies a planned operation without copying snapshot data.
type NodeSyncOperation struct {
	Action NodeSyncAction `json:"action"`
	ItemID string         `json:"itemId"`
}

// NodeSyncPlan is a plan-only result. Applied remains false until a remote protocol
// can safely mutate complete snapshots. RemoteSnapshotFetched identifies the bounded,
// authenticated apply-preview path without exposing snapshot revisions.
type NodeSyncPlan struct {
	NodeID                int                 `json:"nodeId"`
	DryRun                bool                `json:"dryRun"`
	Applied               bool                `json:"applied"`
	RemoteSnapshotFetched bool                `json:"remoteSnapshotFetched,omitempty"`
	Added                 int                 `json:"added"`
	Updated               int                 `json:"updated"`
	Removed               int                 `json:"removed"`
	Operations            []NodeSyncOperation `json:"operations"`
}

// BuildNodeSyncPlan compares local and remote snapshots without performing a sync.
// Operations are ordered by action (add, update, remove) and then item ID.
func BuildNodeSyncPlan(input NodeSyncPlanInput) (NodeSyncPlan, error) {
	if input.NodeID <= 0 {
		return NodeSyncPlan{}, ErrNodeSyncNodeIDRequired
	}
	if !input.Local.Complete || !input.Remote.Complete {
		return NodeSyncPlan{}, ErrNodeSyncSnapshotIncomplete
	}

	local, err := nodeSyncItemsByID(input.Local)
	if err != nil {
		return NodeSyncPlan{}, err
	}
	remote, err := nodeSyncItemsByID(input.Remote)
	if err != nil {
		return NodeSyncPlan{}, err
	}

	plan := NodeSyncPlan{
		NodeID:     input.NodeID,
		DryRun:     input.DryRun,
		Operations: make([]NodeSyncOperation, 0),
	}

	plan.Operations = appendNodeSyncOperations(plan.Operations, NodeSyncAdd, remote, func(itemID string) bool {
		_, exists := local[itemID]
		return !exists
	})
	plan.Operations = appendNodeSyncOperations(plan.Operations, NodeSyncUpdate, remote, func(itemID string) bool {
		localRevision, exists := local[itemID]
		return exists && localRevision != remote[itemID]
	})
	plan.Operations = appendNodeSyncOperations(plan.Operations, NodeSyncRemove, local, func(itemID string) bool {
		_, exists := remote[itemID]
		return !exists
	})

	for _, operation := range plan.Operations {
		switch operation.Action {
		case NodeSyncAdd:
			plan.Added++
		case NodeSyncUpdate:
			plan.Updated++
		case NodeSyncRemove:
			plan.Removed++
		}
	}
	return plan, nil
}

// FetchNodeSyncSnapshot authenticates an operator-supplied token against the
// configured node and fetches only the comparison-safe remote snapshot.
func FetchNodeSyncSnapshot(nodeID int, token string) (NodeSyncSnapshot, error) {
	if nodeID <= 0 {
		return NodeSyncSnapshot{}, ErrNodeSyncNodeIDRequired
	}
	var node model.Node
	if err := database.GetDB().First(&node, nodeID).Error; err != nil {
		if database.IsNotFound(err) {
			return NodeSyncSnapshot{}, ErrNodeSyncNodeNotFound
		}
		return NodeSyncSnapshot{}, ErrNodeSyncRemoteUnavailable
	}
	if !node.Enabled {
		return NodeSyncSnapshot{}, ErrNodeSyncNodeDisabled
	}
	if err := ValidateNodeURLAndName(NodeInput{Name: node.Name, BaseURL: node.BaseURL}); err != nil {
		return NodeSyncSnapshot{}, ErrNodeSyncNodeURLInvalid
	}
	if strings.TrimSpace(token) == "" || bcrypt.CompareHashAndPassword([]byte(node.APITokenHash), []byte(token)) != nil {
		return NodeSyncSnapshot{}, ErrNodeSyncTokenInvalid
	}

	baseURL, _ := url.Parse(strings.TrimSpace(node.BaseURL))
	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + "/api/v2/nodes/sync/snapshot"
	requestCtx, cancel := context.WithTimeout(context.Background(), nodeSyncRequestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, baseURL.String(), nil)
	if err != nil {
		return NodeSyncSnapshot{}, ErrNodeSyncNodeURLInvalid
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	client := &http.Client{Transport: http.DefaultTransport, Timeout: nodeSyncRequestTimeout}
	response, err := client.Do(request)
	if err != nil {
		return NodeSyncSnapshot{}, ErrNodeSyncRemoteUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return NodeSyncSnapshot{}, ErrNodeSyncRemoteRejected
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, nodeSyncMaxResponseBytes+1))
	if err != nil || len(body) > nodeSyncMaxResponseBytes {
		return NodeSyncSnapshot{}, ErrNodeSyncRemoteMalformed
	}
	var envelope struct {
		Success bool             `json:"success"`
		Data    NodeSyncSnapshot `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || !envelope.Success {
		return NodeSyncSnapshot{}, ErrNodeSyncRemoteMalformed
	}
	if !envelope.Data.Complete {
		return NodeSyncSnapshot{}, ErrNodeSyncRemoteIncomplete
	}
	if _, err := nodeSyncItemsByID(envelope.Data); err != nil {
		return NodeSyncSnapshot{}, ErrNodeSyncRemoteMalformed
	}
	return envelope.Data, nil
}

func nodeSyncItemsByID(snapshot NodeSyncSnapshot) (map[string]string, error) {
	items := make(map[string]string, len(snapshot.Items))
	for _, item := range snapshot.Items {
		itemID := strings.TrimSpace(item.ID)
		if itemID == "" {
			return nil, ErrNodeSyncItemIDRequired
		}
		if _, exists := items[itemID]; exists {
			return nil, ErrNodeSyncDuplicateItemID
		}
		items[itemID] = item.Revision
	}
	return items, nil
}

func appendNodeSyncOperations(operations []NodeSyncOperation, action NodeSyncAction, items map[string]string, include func(string) bool) []NodeSyncOperation {
	itemIDs := make([]string, 0, len(items))
	for itemID := range items {
		if include(itemID) {
			itemIDs = append(itemIDs, itemID)
		}
	}
	sort.Strings(itemIDs)
	for _, itemID := range itemIDs {
		operations = append(operations, NodeSyncOperation{Action: action, ItemID: itemID})
	}
	return operations
}
