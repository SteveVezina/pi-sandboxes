package microvm

import (
	"encoding/json"
	"fmt"
)

// ListArtifacts lists guest artifacts through the control plane.
func (c *TransferClient) ListArtifacts() ([]string, error) {
	frame, err := NewArtifactListRequestFrame(c.requestID("artifact-list"), c.sessionID)
	if err != nil {
		return nil, err
	}
	resp, err := c.send(frame)
	if err != nil {
		return nil, err
	}
	var payload ArtifactListPayload
	if err := json.Unmarshal(resp.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode artifact list payload: %w", err)
	}
	return payload.Paths, nil
}

// PullArtifact pulls one artifact through the control plane.
func (c *TransferClient) PullArtifact(path string) ([]byte, uint32, error) {
	frame, err := NewArtifactPullRequestFrame(c.requestID("artifact-pull"), c.sessionID, path)
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.send(frame)
	if err != nil {
		return nil, 0, err
	}
	gotPath, data, mode, err := DecodeFileDataPayload(resp)
	if err != nil {
		return nil, 0, err
	}
	if gotPath != path {
		return nil, 0, fmt.Errorf("artifact response path %q does not match request %q", gotPath, path)
	}
	return data, mode, nil
}
