// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reg

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/yxlib/rpc"
	"github.com/yxlib/yx"
)

const (
	DATA_OPR_TYPE_UPDATE = iota + 1
	DATA_OPR_TYPE_REMOVE
)

const (
	MAX_PUSH_QUE = 10
)

//======================
//     RegObserver
//======================
type RegObserver struct {
	SrvType uint32
	SrvNo   uint32
}

func NewRegObserver(srvType uint32, srvNo uint32) *RegObserver {
	return &RegObserver{
		SrvType: srvType,
		SrvNo:   srvNo,
	}
}

func (o *RegObserver) IsSameObserver(srvType uint32, srvNo uint32) bool {
	return (o.SrvType == srvType && o.SrvNo == srvNo)
}

type RegObserverList = []*RegObserver

//======================
//       Pusher
//======================
type Pusher interface {
	Push(dstPeerType uint32, dstPeerNo uint32, payload ...[]byte) error
}

//======================
//      regCenter
//======================
type regCenter struct {
	info                   *RegInfo
	savePath               string
	bDebug                 bool
	pusher                 Pusher
	mapKey2RegObserverList map[string]RegObserverList
	lckInfoObserver        *sync.RWMutex
	chanOprPush            chan *DataOprPush
	connObserverList       RegObserverList
	lckConnObserver        *sync.RWMutex
	chanConnChange         chan *ConnChangePush
	evtSave                *yx.Event
	logger                 *yx.Logger
	ec                     *yx.ErrCatcher
}

var RegCenter = &regCenter{
	info:                   NewRegInfo(),
	savePath:               "",
	bDebug:                 false,
	pusher:                 nil,
	mapKey2RegObserverList: make(map[string]RegObserverList),
	lckInfoObserver:        &sync.RWMutex{},
	chanOprPush:            make(chan *DataOprPush, MAX_PUSH_QUE),
	connObserverList:       make([]*RegObserver, 0),
	lckConnObserver:        &sync.RWMutex{},
	chanConnChange:         make(chan *ConnChangePush, MAX_PUSH_QUE),
	evtSave:                yx.NewEvent(),
	logger:                 yx.NewLogger("RegCenter"),
	ec:                     yx.NewErrCatcher("RegCenter"),
}

func (c *regCenter) SetSavePath(savePath string) {
	c.savePath = savePath
}

func (c *regCenter) SetDebugMode(bDebug bool) {
	c.bDebug = bDebug
}

func (c *regCenter) SetPusher(p Pusher) {
	c.pusher = p
}

func (c *regCenter) GetRegInfo() *RegInfo {
	return c.info
}

func (c *regCenter) UpdateSrv(srvType uint32, srvNo uint32, bTemp bool, dataBase64 string) {
	var err error = nil
	defer c.ec.Catch("UpdateSrv", &err)

	ok := c.info.HasSrv(srvType, srvNo)
	if !ok {
		err = c.info.AddSrv(srvType, srvNo, bTemp, dataBase64)
	} else {
		err = c.info.SetSrvData(srvType, srvNo, dataBase64)
	}

	if err != nil {
		return
	}

	c.evtSave.Send()

	key := GetSrvKey(srvType, srvNo)
	pushData := NewDataOprPush(KEY_TYPE_SRV_INFO, key, DATA_OPR_TYPE_UPDATE)
	c.chanOprPush <- pushData
	// go s.notifyDataUpdate(key, DATA_OPR_TYPE_UPDATE)
}

func (c *regCenter) RemoveSrv(srvType uint32, srvNo uint32) {
	if c.info.HasSrv(srvType, srvNo) {
		c.info.RemoveSrv(srvType, srvNo)
		c.evtSave.Send()

		key := GetSrvKey(srvType, srvNo)
		pushData := NewDataOprPush(KEY_TYPE_SRV_INFO, key, DATA_OPR_TYPE_REMOVE)
		c.chanOprPush <- pushData
		// go s.notifyDataUpdate(key, DATA_OPR_TYPE_REMOVE)
	}
}

func (c *regCenter) UpdateGlobalData(key string, dataBase64 string) {
	err := c.info.SetGlobalData(key, dataBase64)
	if err != nil {
		c.ec.Catch("UpdateGlobalData", &err)
		return
	}

	c.evtSave.Send()

	pushData := NewDataOprPush(KEY_TYPE_GLOBAL_DATA, key, DATA_OPR_TYPE_UPDATE)
	c.chanOprPush <- pushData
	// go s.notifyDataUpdate(reqData.Key, DATA_OPR_TYPE_UPDATE)
}

func (c *regCenter) RemoveGlobalData(key string) {
	if c.info.HasGlobalData(key) {
		c.info.RemoveGlobalData(key)
		c.evtSave.Send()

		pushData := NewDataOprPush(KEY_TYPE_GLOBAL_DATA, key, DATA_OPR_TYPE_REMOVE)
		c.chanOprPush <- pushData
		// go s.notifyDataUpdate(reqData.Key, DATA_OPR_TYPE_REMOVE)
	}
}

func (c *regCenter) RemoveAllObserverOfSrv(srvType uint32, srvNo uint32) {
	c.removeAllInfoObserverOfSrv(srvType, srvNo)
	c.RemoveConnObserver(srvType, srvNo)
}

func (c *regCenter) NotifyConnChange(srvType uint32, srvNo uint32, connChangeType int) {
	pushData := NewConnChangePush(srvType, srvNo, connChangeType)
	c.chanConnChange <- pushData
}

func (c *regCenter) Start() {
	go c.pushLoop()
	go c.saveLoop()
	// s.BaseService.Start()
}

func (c *regCenter) Stop() {
	// s.BaseService.Stop()
	c.evtSave.Close()
	close(c.chanOprPush)
	close(c.chanConnChange)
}

func (c *regCenter) AddInfoObserver(key string, srvType uint32, srvNo uint32) {
	c.lckInfoObserver.Lock()
	defer c.lckInfoObserver.Unlock()

	list, ok := c.mapKey2RegObserverList[key]
	if !ok {
		list = make([]*RegObserver, 0)
	} else if c.existObserver(list, srvType, srvNo) {
		return
	}

	o := &RegObserver{
		SrvType: srvType,
		SrvNo:   srvNo,
	}

	c.mapKey2RegObserverList[key] = append(list, o)
}

func (c *regCenter) RemoveInfoObserver(key string, srvType uint32, srvNo uint32) {
	c.lckInfoObserver.Lock()
	defer c.lckInfoObserver.Unlock()

	list, ok := c.mapKey2RegObserverList[key]
	if ok {
		c.mapKey2RegObserverList[key] = c.removeObserverFromList(list, srvType, srvNo)
	}
}

func (c *regCenter) removeAllInfoObserverOfSrv(srvType uint32, srvNo uint32) {
	c.lckInfoObserver.Lock()
	defer c.lckInfoObserver.Unlock()

	for key, list := range c.mapKey2RegObserverList {
		c.mapKey2RegObserverList[key] = c.removeObserverFromList(list, srvType, srvNo)
	}
}

func (c *regCenter) cloneInfoObserverList(key string) (RegObserverList, bool) {
	c.lckInfoObserver.RLock()
	defer c.lckInfoObserver.RUnlock()

	list, ok := c.mapKey2RegObserverList[key]
	if !ok {
		return nil, false
	}

	cloneList := make(RegObserverList, len(list))
	copy(cloneList, list)
	return cloneList, ok
}

func (c *regCenter) AddConnObserver(srvType uint32, srvNo uint32) {
	c.lckConnObserver.Lock()
	defer c.lckConnObserver.Unlock()

	if c.existObserver(c.connObserverList, srvType, srvNo) {
		return
	}

	o := &RegObserver{
		SrvType: srvType,
		SrvNo:   srvNo,
	}

	c.connObserverList = append(c.connObserverList, o)
}

func (c *regCenter) RemoveConnObserver(srvType uint32, srvNo uint32) {
	c.lckConnObserver.Lock()
	defer c.lckConnObserver.Unlock()

	c.connObserverList = c.removeObserverFromList(c.connObserverList, srvType, srvNo)
}

func (c *regCenter) cloneConnObserverList() RegObserverList {
	c.lckConnObserver.RLock()
	defer c.lckConnObserver.RUnlock()

	cloneList := make(RegObserverList, len(c.connObserverList))
	copy(cloneList, c.connObserverList)
	return cloneList
}

func (c *regCenter) existObserver(list []*RegObserver, srvType uint32, srvNo uint32) bool {
	for _, observer := range list {
		if observer.IsSameObserver(srvType, srvNo) {
			return true
		}
	}

	return false
}

func (c *regCenter) removeObserverFromList(list []*RegObserver, srvType uint32, srvNo uint32) []*RegObserver {
	for i, observer := range list {
		if observer.IsSameObserver(srvType, srvNo) {
			if len(list) == 1 {
				list = make([]*RegObserver, 0)
			} else {
				list = append(list[:i], list[i+1:]...)
			}
			break
		}
	}

	return list
}

func (c *regCenter) pushLoop() {
	for {
		select {
		case pushData, ok := <-c.chanOprPush:
			if !ok {
				goto Exit0
			}

			c.notifyDataUpdate(pushData)

		case pushData, ok := <-c.chanConnChange:
			if !ok {
				goto Exit0
			}

			c.notifyConnChange(pushData)
		}
	}

Exit0:
	return
}

func (c *regCenter) notifyDataUpdate(pushData *DataOprPush) {
	list, ok := c.cloneInfoObserverList(pushData.Key)
	if ok {
		c.push(pushData, DATA_OPR_PUSH_FUNC_NO, list)
	}

	idx := strings.LastIndex(pushData.Key, "/")
	if idx <= 0 {
		return
	}

	parentKey := pushData.Key[:idx]
	list, ok = c.cloneInfoObserverList(parentKey)
	if ok {
		c.push(pushData, DATA_OPR_PUSH_FUNC_NO, list)
	}
}

// func (s *Server) pushDataUpdate(key string, opr int, list []*RegObserver) {
// 	if len(list) == 0 {
// 		return
// 	}

// 	pushData := &DataOprPush{
// 		Key:     key,
// 		Operate: opr,
// 	}

// 	s.push(pushData, list)
// }

func (c *regCenter) notifyConnChange(pushData *ConnChangePush) {
	list := c.cloneConnObserverList()
	c.push(pushData, CONN_CHANGE_FUNC_NO, list)
}

func (c *regCenter) push(pushData interface{}, funcNo uint16, list []*RegObserver) {
	if len(list) == 0 {
		return
	}

	if c.pusher == nil {
		c.logger.E("pusher is nil")
		return
	}

	packData, err := json.Marshal(pushData)
	if err != nil {
		c.logger.E("push json.Marshal err: ", err)
		return
	}

	h := rpc.NewPackHeader(PUSH_MARK, 0, funcNo)
	headerData, err := h.Marshal()
	if err != nil {
		c.logger.E("push PackHeader.Marshal err: ", err)
		return
	}

	payload := make([]rpc.ByteArray, 0, 2)
	payload = append(payload, headerData, packData)

	for _, observer := range list {
		c.pusher.Push(observer.SrvType, observer.SrvNo, payload...)
	}
}

func (c *regCenter) saveLoop() {
	for {
		// _, ok := <-s.evtSave.C
		// if !ok {
		// 	break
		// }

		err := c.evtSave.Wait()
		if err != nil {
			break
		}

		c.info.Save(c.savePath)
		if c.bDebug {
			c.info.Dump()
		}
	}
}
