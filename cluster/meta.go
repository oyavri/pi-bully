package cluster

import (
	"encoding/json"
	"fmt"
)

type NodeMeta struct {
	GRPCAddr string `json:"grpc_addr"`
}

func encodeNodeMeta(m NodeMeta) ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode node meta: %w", err)
	}

	return b, nil
}

func decodeNodeMeta(b []byte) (NodeMeta, error) {
	var m NodeMeta
	if err := json.Unmarshal(b, &m); err != nil {
		return NodeMeta{}, fmt.Errorf("decode node meta: %w", err)
	}

	return m, nil
}
