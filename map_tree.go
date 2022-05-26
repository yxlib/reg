// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reg

import "errors"

var (
	ErrMTChildIsNil     = errors.New("child is nil")
	ErrMTChildExists    = errors.New("child exists")
	ErrMTChildNotExists = errors.New("child not exists")
)

type MapTreeNode struct {
	mapKey2Child map[string]*MapTreeNode
	nodeData     interface{}
}

func NewMapTreeNode() *MapTreeNode {
	return &MapTreeNode{
		mapKey2Child: make(map[string]*MapTreeNode),
		nodeData:     nil,
	}
}

func (n *MapTreeNode) AddChild(key string, child *MapTreeNode) error {
	if child == nil {
		return ErrMTChildIsNil
	}

	if n.HasChild(key) {
		return ErrMTChildExists
	}

	n.mapKey2Child[key] = child
	return nil
}

func (n *MapTreeNode) HasChild(key string) bool {
	_, ok := n.mapKey2Child[key]
	return ok
}

func (n *MapTreeNode) GetChild(key string) (*MapTreeNode, bool) {
	child, ok := n.mapKey2Child[key]
	return child, ok
}

func (n *MapTreeNode) RemoveChild(key string) error {
	if !n.HasChild(key) {
		return ErrMTChildNotExists
	}

	delete(n.mapKey2Child, key)
	return nil
}

func (n *MapTreeNode) AllChildKeys() []string {
	keys := make([]string, 0, len(n.mapKey2Child))
	for k := range n.mapKey2Child {
		keys = append(keys, k)
	}

	return keys
}

func (n *MapTreeNode) AllChilds() []*MapTreeNode {
	childs := make([]*MapTreeNode, 0, len(n.mapKey2Child))
	for _, v := range n.mapKey2Child {
		childs = append(childs, v)
	}

	return childs
}

func (n *MapTreeNode) SetData(d interface{}) {
	n.nodeData = d
}

func (n *MapTreeNode) GetData() interface{} {
	return n.nodeData
}

type MapTree struct {
	root *MapTreeNode
}

func NewMapTree() *MapTree {
	return &MapTree{
		root: NewMapTreeNode(),
	}
}

func (t *MapTree) GetRoot() *MapTreeNode {
	return t.root
}
