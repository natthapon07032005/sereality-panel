package service

import "errors"

type APIV2Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIV2Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIV2Error `json:"error,omitempty"`
}

func NewAPIV2Response(data interface{}, err error) APIV2Response {
	if err != nil {
		return APIV2Response{Success: false, Error: &APIV2Error{Code: "request_failed", Message: "request could not be completed"}}
	}
	return APIV2Response{Success: true, Data: data}
}

var ErrNodeSyncUnavailable = errors.New("node sync transport is not configured")

type NodeSyncResult struct {
	NodeID  int `json:"nodeId"`
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Removed int `json:"removed"`
}

func BuildNodeSyncResult(nodeID int, localItems, remoteItems []string) NodeSyncResult {
	local := make(map[string]struct{}, len(localItems))
	remote := make(map[string]struct{}, len(remoteItems))
	for _, item := range localItems {
		local[item] = struct{}{}
	}
	for _, item := range remoteItems {
		remote[item] = struct{}{}
	}
	result := NodeSyncResult{NodeID: nodeID}
	for item := range remote {
		if _, ok := local[item]; ok {
			result.Updated++
		} else {
			result.Added++
		}
	}
	for item := range local {
		if _, ok := remote[item]; !ok {
			result.Removed++
		}
	}
	return result
}
