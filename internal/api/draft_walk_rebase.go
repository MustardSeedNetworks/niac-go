package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

func (s *Server) rebaseCapturedDraftContent(content string) (string, error) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", fmt.Errorf("parse draft for captured walk rebasing: %w", err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return content, nil
	}
	root := document.Content[0]
	previousRoot := mappingScalar(root, "include_path")
	walkRoot := s.library.SubDir(library.KindWalks)
	state := walkRebaseState{
		previousRoot: previousRoot, walkRoot: walkRoot, contentLibrary: s.library,
	}
	normalizeCapturedWalks(&document, &state)
	if !state.foundCaptured {
		return content, nil
	}
	if state.hasIncompatibleWalk {
		return "", errors.New("captured walks cannot be combined with walks outside the content library")
	}
	setMappingScalar(root, "include_path", walkRoot)
	rebased, err := yaml.Marshal(&document)
	if err != nil {
		return "", fmt.Errorf("serialize rebased captured walk draft: %w", err)
	}
	return string(rebased), nil
}

type walkRebaseState struct {
	previousRoot        string
	walkRoot            string
	contentLibrary      *library.Library
	foundCaptured       bool
	hasIncompatibleWalk bool
}

func normalizeCapturedWalks(node *yaml.Node, state *walkRebaseState) {
	if node.Kind == yaml.MappingNode {
		normalizeCapturedMapping(node, state)
	}
	for _, child := range node.Content {
		normalizeCapturedWalks(child, state)
	}
}

func normalizeCapturedMapping(mapping *yaml.Node, state *walkRebaseState) {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		key, value := mapping.Content[index], mapping.Content[index+1]
		if key.Value == "walk_file" {
			normalizeCapturedWalkScalar(value, state)
		}
		if key.Value == "walk_files" && value.Kind == yaml.SequenceNode {
			for _, item := range value.Content {
				normalizeCapturedWalkScalar(item, state)
			}
		}
	}
}

func normalizeCapturedWalkScalar(node *yaml.Node, state *walkRebaseState) {
	if node.Kind != yaml.ScalarNode {
		return
	}
	value := filepath.ToSlash(node.Value)
	if strings.HasPrefix(value, "captured/") {
		node.Value = value
		state.foundCaptured = true
		return
	}
	if !filepath.IsAbs(node.Value) {
		if normalizeLibraryWalk(node, state, node.Value) {
			return
		}
		if !filepath.IsAbs(state.previousRoot) {
			state.hasIncompatibleWalk = true
			return
		}
		node.Value = filepath.Join(state.previousRoot, node.Value)
		state.hasIncompatibleWalk = true
		return
	}
	if state.previousRoot != "" {
		relative, err := filepath.Rel(state.previousRoot, node.Value)
		if err == nil && strings.HasPrefix(filepath.ToSlash(relative), "captured/") {
			node.Value = filepath.ToSlash(relative)
			state.foundCaptured = true
			return
		}
		if err == nil && normalizeLibraryWalk(node, state, relative) {
			return
		}
	}
	normalizeWalkUnderRoot(node, state)
}

func normalizeWalkUnderRoot(node *yaml.Node, state *walkRebaseState) {
	relative, err := filepath.Rel(state.walkRoot, node.Value)
	if err != nil || !normalizeLibraryWalk(node, state, relative) {
		state.hasIncompatibleWalk = true
	}
}

func normalizeLibraryWalk(node *yaml.Node, state *walkRebaseState, relative string) bool {
	relative = filepath.ToSlash(relative)
	if filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, "../") {
		return false
	}
	if _, err := state.contentLibrary.ReadFile(library.KindWalks, relative); err != nil {
		return false
	}
	node.Value = relative
	return true
}

func mappingScalar(mapping *yaml.Node, key string) string {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key && mapping.Content[index+1].Kind == yaml.ScalarNode {
			return mapping.Content[index+1].Value
		}
	}
	return ""
}

func setMappingScalar(mapping *yaml.Node, key, value string) {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			mapping.Content[index+1].Kind = yaml.ScalarNode
			mapping.Content[index+1].Tag = "!!str"
			mapping.Content[index+1].Value = value
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}
