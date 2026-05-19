// Copyright 2026. Triad National Security, LLC. All rights reserved.

package pftool

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	ConduitPrefix = "#CONDUIT-MSG"
)

type PFTMessage struct {
	Type string `json:"Type"`
}

func (pm *PFTMessage) GetType() (PFTMessageType, error) {
	if val, ok := PFTMessageType_value[pm.Type]; ok {
		return PFTMessageType(val), nil
	} else {
		return PFTMessageType_NONE, fmt.Errorf("unrecognized message type: %v", pm.Type)
	}
}

type PFTError struct {
	Class   string `json:"Class,omitempty"`
	Origin  string `json:"Origin,omitempty"`
	Errno   int    `json:"Errno,omitempty"`
	Message string `json:"Message,omitempty"`
}

func parsePFTError(b []byte) (*PFTError, error) {
	pm := &PFTError{}
	err := json.Unmarshal(b, pm)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %v %v", err, string(b))
	}
	pm.Class = strings.TrimSpace(pm.Class)
	pm.Origin = strings.TrimSpace(pm.Origin)
	pm.Message = strings.TrimSpace(pm.Message)

	return pm, nil
}

type PFTHeader struct {
	SourceFS      string `json:"SourceFS,omitempty"`
	DestinationFS string `json:"DestinationFS,omitempty"`
}

func parsePFTHeader(b []byte) (*PFTHeader, error) {
	pm := &PFTHeader{}
	err := json.Unmarshal(b, pm)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %v %v", err, string(b))
	}
	pm.SourceFS = strings.TrimSpace(pm.SourceFS)
	pm.DestinationFS = strings.TrimSpace(pm.DestinationFS)

	return pm, nil
}

type PFTFooter struct {
	FilesChunks int    `json:"Files&Chunks,omitempty"`
	Data        string `json:"Data,omitempty"`
	Bandwidth   string `json:"Bandwidth,omitempty"`
	Files       int    `json:"Files,omitempty"`
	Directories int    `json:"Directories,omitempty"`
}

func parsePFTFooter(b []byte) (*PFTFooter, error) {
	pm := &PFTFooter{}
	err := json.Unmarshal(b, pm)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %v %v", err, string(b))
	}
	pm.Data = strings.TrimSpace(pm.Data)
	pm.Bandwidth = strings.TrimSpace(pm.Bandwidth)

	return pm, nil
}

type PFTAccum struct {
	FilesChunks  int    `json:"Files&Chunks,omitempty"`
	DataFinished string `json:"DataFinished,omitempty"`
	Bandwidth    string `json:"Bandwidth,omitempty"`
}

func parsePFTAccum(b []byte) (*PFTAccum, error) {
	pm := &PFTAccum{}
	err := json.Unmarshal(b, pm)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %v %v", err, string(b))
	}
	pm.DataFinished = strings.TrimSpace(pm.DataFinished)
	pm.Bandwidth = strings.TrimSpace(pm.Bandwidth)

	return pm, nil
}

type PFTNoMove struct {
	Path string `json:"Path,omitempty"`
}

func parsePFTNoMove(b []byte) (*PFTNoMove, error) {
	pm := &PFTNoMove{}
	err := json.Unmarshal(b, pm)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %v %v", err, string(b))
	}
	pm.Path = strings.TrimSpace(pm.Path)

	return pm, nil
}

type PFTMessageType int32

const (
	PFTMessageType_NONE   PFTMessageType = 0
	PFTMessageType_HEADER PFTMessageType = 1
	PFTMessageType_FOOTER PFTMessageType = 2
	PFTMessageType_ERROR  PFTMessageType = 3
	PFTMessageType_ACCUM  PFTMessageType = 4
	PFTMessageType_NOMOVE PFTMessageType = 5
)

// Enum value maps for TransferState.
var (
	PFTMessageType_name = map[int32]string{
		0: "NONE",
		1: "HEADER",
		2: "FOOTER",
		3: "ERROR",
		4: "ACCUM",
		5: "NOMOVE",
	}
	PFTMessageType_value = map[string]int32{
		"NONE":   0,
		"HEADER": 1,
		"FOOTER": 2,
		"ERROR":  3,
		"ACCUM":  4,
		"NOMOVE": 5,
	}
)

func getPFToolMessageType(message []byte) (PFTMessageType, error) {
	pm := &PFTMessage{}
	err := json.Unmarshal(message, pm)
	if err != nil {
		return PFTMessageType_NONE, fmt.Errorf("failed to unmarshal message: %v %v", err, string(message))
	}
	t, err := pm.GetType()
	if err != nil {
		return PFTMessageType_NONE, err
	}
	return t, nil
}
