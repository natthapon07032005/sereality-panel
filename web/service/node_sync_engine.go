package service

import (
	"errors"
	"sort"
	"strings"
)

var (
	ErrNodeSyncNodeIDRequired     = errors.New("node ID is required")
	ErrNodeSyncItemIDRequired     = errors.New("node sync item ID is required")
	ErrNodeSyncDuplicateItemID    = errors.New("node sync item ID must be unique within a snapshot")
	ErrNodeSyncSnapshotIncomplete = errors.New("node sync snapshot is incomplete")
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

// NodeSyncPlan is a plan-only result. Applied is always false because this package
// does not contact, mutate, or otherwise execute work on a remote node.
type NodeSyncPlan struct {
	NodeID     int                 `json:"nodeId"`
	DryRun     bool                `json:"dryRun"`
	Applied    bool                `json:"applied"`
	Added      int                 `json:"added"`
	Updated    int                 `json:"updated"`
	Removed    int                 `json:"removed"`
	Operations []NodeSyncOperation `json:"operations"`
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
