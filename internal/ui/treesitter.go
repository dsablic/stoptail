package ui

import (
	sitter "github.com/smacker/go-tree-sitter"
	tree_sitter_json "github.com/tree-sitter/tree-sitter-json/bindings/go"
)

var jsonLanguage = sitter.NewLanguage(tree_sitter_json.Language())
