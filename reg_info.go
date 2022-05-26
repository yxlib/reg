// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	ErrSrvNotExists = errors.New("server not exists")
	ErrEmptyPath    = errors.New("empty path")
)

type SrvInfo struct {
	SrvType    uint16 `json:"type"`
	SrvNo      uint16 `json:"no"`
	IsTemp     bool   `json:"bTemp"`
	DataBase64 string `json:"data"`
}

type RegSavedInfo struct {
	SrvInfos          []*SrvInfo        `json:"srv"`
	MapGlobalKey2Data map[string]string `json:"global"`
}

type RegInfo struct {
	treeSrvInfos    *MapTree
	treeGlobalInfos *MapTree
}

func NewRegInfo() *RegInfo {
	return &RegInfo{
		treeSrvInfos:    NewMapTree(),
		treeGlobalInfos: NewMapTree(),
	}
}

func (r *RegInfo) AddSrv(srvType uint16, srvNo uint16, bTemp bool, dataBase64 string) error {
	info := &SrvInfo{
		SrvType:    srvType,
		SrvNo:      srvNo,
		IsTemp:     bTemp,
		DataBase64: dataBase64,
	}

	key := r.GetSrvKey(srvType, srvNo)
	err := r.setData(r.treeSrvInfos, key, info)
	return err
}

func (r *RegInfo) RemoveSrv(srvType uint16, srvNo uint16) {
	key := r.GetSrvKey(srvType, srvNo)
	r.removeData(r.treeSrvInfos, key)
}

func (r *RegInfo) IsTempSrv(srvType uint16, srvNo uint16) (bool, error) {
	key := r.GetSrvKey(srvType, srvNo)
	data, ok := r.getData(r.treeSrvInfos, key)
	if !ok {
		return false, ErrSrvNotExists
	}

	info := data.(*SrvInfo)
	return info.IsTemp, nil
}

func (r *RegInfo) HasSrv(srvType uint16, srvNo uint16) bool {
	key := r.GetSrvKey(srvType, srvNo)
	_, ok := r.getData(r.treeSrvInfos, key)
	return ok
}

func (r *RegInfo) GetSrvInfo(srvType uint16, srvNo uint16) (*SrvInfo, bool) {
	key := r.GetSrvKey(srvType, srvNo)
	data, ok := r.getData(r.treeSrvInfos, key)
	if !ok {
		return nil, false
	}

	info := data.(*SrvInfo)
	return info, true
}

func (r *RegInfo) GetSrvData(srvType uint16, srvNo uint16) (string, bool) {
	info, ok := r.GetSrvInfo(srvType, srvNo)
	return info.DataBase64, ok
}

func (r *RegInfo) SetSrvData(srvType uint16, srvNo uint16, dataBase64 string) error {
	key := r.GetSrvKey(srvType, srvNo)
	d, ok := r.getData(r.treeSrvInfos, key)
	if !ok {
		return ErrSrvNotExists
	}

	info := d.(*SrvInfo)
	info.DataBase64 = dataBase64
	return nil
}

func (r *RegInfo) GetAllSrvNos(srvType uint16) ([]uint16, bool) {
	key := r.GetSrvTypeKey(srvType)
	node, ok := r.getNode(r.treeSrvInfos, key)
	if !ok {
		return nil, false
	}

	childKeys := node.AllChildKeys()
	srvNos := make([]uint16, 0, len(childKeys))
	for _, childKey := range childKeys {
		no, _ := strconv.ParseUint(childKey, 10, 16)
		srvNos = append(srvNos, uint16(no))
	}

	return srvNos, true
}

func (r *RegInfo) GetAllSrvInfos(srvType uint16) ([]*SrvInfo, bool) {
	key := r.GetSrvTypeKey(srvType)
	node, ok := r.getNode(r.treeSrvInfos, key)
	if !ok {
		return nil, false
	}

	childs := node.AllChilds()
	srvInfos := make([]*SrvInfo, 0, len(childs))
	for _, child := range childs {
		d := child.GetData()
		info := d.(*SrvInfo)
		srvInfos = append(srvInfos, info)
	}

	return srvInfos, true
}

func (r *RegInfo) SetGlobalData(key string, data string) error {
	return r.setData(r.treeGlobalInfos, key, data)
}

func (r *RegInfo) GetGlobalData(key string) (string, bool) {
	data, ok := r.getData(r.treeGlobalInfos, key)
	if !ok || data == nil {
		return "", false
	}

	return data.(string), ok
}

func (r *RegInfo) HasGlobalData(key string) bool {
	_, ok := r.getData(r.treeGlobalInfos, key)
	return ok
}

func (r *RegInfo) RemoveGlobalData(key string) {
	r.removeData(r.treeGlobalInfos, key)
}

func (r *RegInfo) Load(filePath string) error {
	// open file
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}

	defer f.Close()

	// unmarshal json
	savedInfo := &RegSavedInfo{
		SrvInfos:          make([]*SrvInfo, 0),
		MapGlobalKey2Data: make(map[string]string),
	}

	err = json.NewDecoder(f).Decode(savedInfo)
	if err != nil {
		return err
	}

	// unmarshal server informations
	for _, srvInfo := range savedInfo.SrvInfos {
		r.AddSrv(srvInfo.SrvType, srvInfo.SrvNo, srvInfo.IsTemp, srvInfo.DataBase64)
	}

	// unmarshal global informations
	for key, val := range savedInfo.MapGlobalKey2Data {
		r.SetGlobalData(key, val)
	}

	return nil
}

func (r *RegInfo) Save(filePath string) error {
	savedInfo := &RegSavedInfo{
		SrvInfos:          make([]*SrvInfo, 0),
		MapGlobalKey2Data: make(map[string]string),
	}

	r.marshalSrvInfos(savedInfo)
	r.marshalGlobalInfos(savedInfo)

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	defer f.Close()

	err = json.NewEncoder(f).Encode(savedInfo)
	return err
}

func (r *RegInfo) GetSrvTypeKey(srvType uint16) string {
	return fmt.Sprintf("/%d", srvType)
}

func (r *RegInfo) GetSrvKey(srvType uint16, srvNo uint16) string {
	return fmt.Sprintf("/%d/%d", srvType, srvNo)
}

func (r *RegInfo) setData(tree *MapTree, key string, data interface{}) error {
	subPaths := r.parseInfoPath(key)
	if len(subPaths) == 0 {
		return ErrEmptyPath
	}

	var parent *MapTreeNode = nil
	ok := false
	child := tree.root
	for _, subPath := range subPaths {
		parent = child
		child, ok = parent.GetChild(subPath)
		if !ok {
			child = NewMapTreeNode()
			parent.AddChild(subPath, child)
		}
	}

	child.SetData(data)
	return nil
}

func (r *RegInfo) getData(tree *MapTree, key string) (interface{}, bool) {
	node, ok := r.getNode(tree, key)
	if !ok {
		return nil, false
	}

	return node.GetData(), true
}

func (r *RegInfo) removeData(tree *MapTree, key string) {
	subPaths := r.parseInfoPath(key)
	if len(subPaths) == 0 {
		return
	}

	ok := false
	node := tree.root
	for i, subPath := range subPaths {
		if i == len(subPaths)-1 {
			node.RemoveChild(subPath)
			break
		}

		node, ok = node.GetChild(subPath)
		if !ok {
			break
		}
	}
}

func (r *RegInfo) getNode(tree *MapTree, key string) (*MapTreeNode, bool) {
	subPaths := r.parseInfoPath(key)
	if len(subPaths) == 0 {
		return nil, false
	}

	ok := false
	node := tree.root
	for _, subPath := range subPaths {
		node, ok = node.GetChild(subPath)
		if !ok {
			return nil, false
		}
	}

	return node, true
}

func (r *RegInfo) parseInfoPath(path string) []string {
	subPaths := make([]string, 0)

	if len(path) <= 1 {
		return subPaths
	}

	idx := strings.Index(path, "/")
	if idx != 0 {
		return subPaths
	}

	// subPath = append(subPath, "/")
	// if len(path) == 1 {
	// 	return subPath
	// }

	subStr := strings.Split(path[1:], "/")
	return append(subPaths, subStr...)
}

func (r *RegInfo) marshalSrvInfos(savedInfo *RegSavedInfo) {
	r.visitSaveSrvInfos(savedInfo, "", r.treeSrvInfos.root)
}

func (r *RegInfo) visitSaveSrvInfos(savedInfo *RegSavedInfo, parentPath string, parentNode *MapTreeNode) {
	if parentNode == nil {
		return
	}

	d := parentNode.GetData()
	if d != nil {
		info := d.(*SrvInfo)
		savedInfo.SrvInfos = append(savedInfo.SrvInfos, info)
	}

	childKeys := parentNode.AllChildKeys()
	for _, key := range childKeys {
		path := parentPath + "/" + key
		childNode, _ := parentNode.GetChild(key)
		r.visitSaveSrvInfos(savedInfo, path, childNode)
	}
}

func (r *RegInfo) marshalGlobalInfos(savedInfo *RegSavedInfo) {
	r.visitSaveGlobalInfos(savedInfo, "", r.treeGlobalInfos.root)
}

func (r *RegInfo) visitSaveGlobalInfos(savedInfo *RegSavedInfo, parentPath string, parentNode *MapTreeNode) {
	if parentNode == nil {
		return
	}

	d := parentNode.GetData()
	if d != nil {
		data := d.(string)
		savedInfo.MapGlobalKey2Data[parentPath] = data
	}

	childKeys := parentNode.AllChildKeys()
	for _, key := range childKeys {
		path := parentPath + "/" + key
		childNode, _ := parentNode.GetChild(key)
		r.visitSaveGlobalInfos(savedInfo, path, childNode)
	}
}
