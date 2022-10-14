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

type Server struct {
	*rpc.BaseService
	info                   *RegInfo
	savePath               string
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

func NewServer(net rpc.Net, savePath string) *Server {
	s := &Server{
		BaseService:            rpc.NewBaseService(net),
		info:                   NewRegInfo(),
		savePath:               savePath,
		mapKey2RegObserverList: make(map[string]RegObserverList),
		lckInfoObserver:        &sync.RWMutex{},
		chanOprPush:            make(chan *DataOprPush, MAX_PUSH_QUE),
		connObserverList:       make([]*RegObserver, 0),
		lckConnObserver:        &sync.RWMutex{},
		chanConnChange:         make(chan *ConnChangePush, MAX_PUSH_QUE),
		evtSave:                yx.NewEvent(),
		logger:                 yx.NewLogger("reg.Server"),
		ec:                     yx.NewErrCatcher("reg.Server"),
	}

	s.SetName(REG_SRV)
	s.SetInterceptor(&rpc.JsonServInterceptor{})
	return s
}

func (s *Server) GetRegInfo() *RegInfo {
	return s.info
}

// func (s *Server) Save() {
// 	s.evtSave.Send()
// }

func (s *Server) UpdateSrv(srvType uint32, srvNo uint32, bTemp bool, dataBase64 string) {
	var err error = nil
	defer s.ec.Catch("UpdateSrv", &err)

	ok := s.info.HasSrv(srvType, srvNo)
	if !ok {
		err = s.info.AddSrv(srvType, srvNo, bTemp, dataBase64)
	} else {
		err = s.info.SetSrvData(srvType, srvNo, dataBase64)
	}

	if err != nil {
		return
	}

	s.evtSave.Send()

	key := GetSrvKey(srvType, srvNo)
	pushData := NewDataOprPush(KEY_TYPE_SRV_INFO, key, DATA_OPR_TYPE_UPDATE)
	s.chanOprPush <- pushData
	// go s.notifyDataUpdate(key, DATA_OPR_TYPE_UPDATE)
}

func (s *Server) RemoveSrv(srvType uint32, srvNo uint32) {
	if s.info.HasSrv(srvType, srvNo) {
		s.info.RemoveSrv(srvType, srvNo)
		s.evtSave.Send()

		key := GetSrvKey(srvType, srvNo)
		pushData := NewDataOprPush(KEY_TYPE_SRV_INFO, key, DATA_OPR_TYPE_REMOVE)
		s.chanOprPush <- pushData
		// go s.notifyDataUpdate(key, DATA_OPR_TYPE_REMOVE)
	}
}

func (s *Server) UpdateGlobalData(key string, dataBase64 string) {
	err := s.info.SetGlobalData(key, dataBase64)
	if err != nil {
		s.ec.Catch("UpdateGlobalData", &err)
		return
	}

	s.evtSave.Send()

	pushData := NewDataOprPush(KEY_TYPE_GLOBAL_DATA, key, DATA_OPR_TYPE_UPDATE)
	s.chanOprPush <- pushData
	// go s.notifyDataUpdate(reqData.Key, DATA_OPR_TYPE_UPDATE)
}

func (s *Server) RemoveGlobalData(key string) {
	if s.info.HasGlobalData(key) {
		s.info.RemoveGlobalData(key)
		s.evtSave.Send()

		pushData := NewDataOprPush(KEY_TYPE_GLOBAL_DATA, key, DATA_OPR_TYPE_REMOVE)
		s.chanOprPush <- pushData
		// go s.notifyDataUpdate(reqData.Key, DATA_OPR_TYPE_REMOVE)
	}
}

func (s *Server) RemoveAllObserverOfSrv(srvType uint32, srvNo uint32) {
	s.removeAllInfoObserverOfSrv(srvType, srvNo)
	s.removeConnObserver(srvType, srvNo)
}

func (s *Server) NotifyConnChange(srvType uint32, srvNo uint32, connChangeType int) {
	pushData := NewConnChangePush(srvType, srvNo, connChangeType)
	s.chanConnChange <- pushData
}

func (s *Server) Start() {
	go s.pushLoop()
	go s.saveLoop()
	s.BaseService.Start()
}

func (s *Server) Stop() {
	s.BaseService.Stop()
	s.evtSave.Close()
	close(s.chanOprPush)
	close(s.chanConnChange)
}

func (s *Server) OnUpdateSrv(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*UpdateSrvReq)
	s.UpdateSrv(reqData.SrvType, reqData.SrvNo, reqData.IsTemp, reqData.DataBase64)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnRemoveSrv(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*RemoveSrvReq)
	s.RemoveSrv(reqData.SrvType, reqData.SrvNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnGetSrv(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*GetSrvReq)
	respData := resp.(*GetSrvResp)

	srvInfo, ok := s.info.GetSrvInfo(reqData.SrvType, reqData.SrvNo)
	if ok {
		respData.SetResult(RES_CODE_SUCC, "")
	} else {
		respData.SetResult(RES_CODE_SRV_NOT_EXISTS, "server not exists")
	}

	respData.Data = srvInfo
	return nil
}

func (s *Server) OnGetSrvByKey(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*GetSrvByKeyReq)
	respData := resp.(*GetSrvByKeyResp)

	srvInfo, ok := s.info.GetSrvInfoByKey(reqData.Key)
	if ok {
		respData.SetResult(RES_CODE_SUCC, "")
	} else {
		respData.SetResult(RES_CODE_SRV_NOT_EXISTS, "server not exists")
	}

	respData.Data = srvInfo
	return nil
}

func (s *Server) OnGetSrvsByType(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*GetSrvsByTypeReq)
	respData := resp.(*GetSrvsByTypeResp)

	infos, ok := s.info.GetAllSrvInfos(reqData.SrvType)
	if ok {
		respData.SetResult(RES_CODE_SUCC, "")
	} else {
		respData.SetResult(RES_CODE_SRV_TYPE_NOT_EXISTS, "server type not exists")
	}

	respData.Data = infos
	return nil
}

func (s *Server) OnWatchSrv(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*WatchSrvReq)
	key := GetSrvKey(reqData.SrvType, reqData.SrvNo)
	s.addInfoObserver(key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopWatchSrv(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*StopWatchSrvReq)
	key := GetSrvKey(reqData.SrvType, reqData.SrvNo)
	s.removeInfoObserver(key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnWatchSrvsByType(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*WatchSrvsByTypeReq)
	key := GetSrvTypeKey(reqData.SrvType)
	s.addInfoObserver(key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopWatchSrvsByType(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*StopWatchSrvsByTypeReq)
	key := GetSrvTypeKey(reqData.SrvType)
	s.removeInfoObserver(key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnUpdateGlobalData(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*UpdateGlobalDataReq)
	s.UpdateGlobalData(reqData.Key, reqData.DataBase64)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnRemoveGlobalData(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*RemoveGlobalDataReq)
	s.RemoveGlobalData(reqData.Key)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnGetGlobalData(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*GetGlobalDataReq)
	respData := resp.(*GetGlobalDataResp)

	dataBase64, ok := s.info.GetGlobalData(reqData.Key)
	if ok {
		respData.SetResult(RES_CODE_SUCC, "")
	} else {
		respData.SetResult(RES_CODE_GLOBAL_DATA_NOT_EXISTS, "global data not exists")
	}

	respData.DataBase64 = dataBase64
	return nil
}

func (s *Server) OnWatchGlobalData(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*WatchGlobalDataReq)
	s.addInfoObserver(reqData.Key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopWatchGlobalData(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*StopWatchGlobalDataReq)
	s.removeInfoObserver(reqData.Key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnWatchConn(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	// reqData := req.(*WatchConnReq)
	s.addConnObserver(srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopWatchConn(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	// reqData := req.(*StopWatchConnReq)
	s.removeConnObserver(srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopAllWatch(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*StopAllWatchReq)
	s.RemoveAllObserverOfSrv(reqData.SrvType, reqData.SrvNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) addInfoObserver(key string, srvType uint32, srvNo uint32) {
	s.lckInfoObserver.Lock()
	defer s.lckInfoObserver.Unlock()

	list, ok := s.mapKey2RegObserverList[key]
	if !ok {
		list = make([]*RegObserver, 0)
	} else if s.existObserver(list, srvType, srvNo) {
		return
	}

	o := &RegObserver{
		SrvType: srvType,
		SrvNo:   srvNo,
	}

	s.mapKey2RegObserverList[key] = append(list, o)
}

func (s *Server) removeInfoObserver(key string, srvType uint32, srvNo uint32) {
	s.lckInfoObserver.Lock()
	defer s.lckInfoObserver.Unlock()

	list, ok := s.mapKey2RegObserverList[key]
	if ok {
		s.mapKey2RegObserverList[key] = s.removeObserverFromList(list, srvType, srvNo)
	}
}

func (s *Server) removeAllInfoObserverOfSrv(srvType uint32, srvNo uint32) {
	s.lckInfoObserver.Lock()
	defer s.lckInfoObserver.Unlock()

	for key, list := range s.mapKey2RegObserverList {
		s.mapKey2RegObserverList[key] = s.removeObserverFromList(list, srvType, srvNo)
	}
}

func (s *Server) cloneInfoObserverList(key string) (RegObserverList, bool) {
	s.lckInfoObserver.RLock()
	defer s.lckInfoObserver.RUnlock()

	list, ok := s.mapKey2RegObserverList[key]
	if !ok {
		return nil, false
	}

	cloneList := make(RegObserverList, len(list))
	copy(cloneList, list)
	return cloneList, ok
}

func (s *Server) addConnObserver(srvType uint32, srvNo uint32) {
	s.lckConnObserver.Lock()
	defer s.lckConnObserver.Unlock()

	if s.existObserver(s.connObserverList, srvType, srvNo) {
		return
	}

	o := &RegObserver{
		SrvType: srvType,
		SrvNo:   srvNo,
	}

	s.connObserverList = append(s.connObserverList, o)
}

func (s *Server) removeConnObserver(srvType uint32, srvNo uint32) {
	s.lckConnObserver.Lock()
	defer s.lckConnObserver.Unlock()

	s.connObserverList = s.removeObserverFromList(s.connObserverList, srvType, srvNo)
}

func (s *Server) cloneConnObserverList() RegObserverList {
	s.lckConnObserver.RLock()
	defer s.lckConnObserver.RUnlock()

	cloneList := make(RegObserverList, len(s.connObserverList))
	copy(cloneList, s.connObserverList)
	return cloneList
}

func (s *Server) existObserver(list []*RegObserver, srvType uint32, srvNo uint32) bool {
	for _, observer := range list {
		if observer.IsSameObserver(srvType, srvNo) {
			return true
		}
	}

	return false
}

func (s *Server) removeObserverFromList(list []*RegObserver, srvType uint32, srvNo uint32) []*RegObserver {
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

func (s *Server) pushLoop() {
	for {
		select {
		case pushData, ok := <-s.chanOprPush:
			if !ok {
				goto Exit0
			}

			s.notifyDataUpdate(pushData)

		case pushData, ok := <-s.chanConnChange:
			if !ok {
				goto Exit0
			}

			s.notifyConnChange(pushData)
		}
	}

Exit0:
	return
}

func (s *Server) notifyDataUpdate(pushData *DataOprPush) {
	list, ok := s.cloneInfoObserverList(pushData.Key)
	if ok {
		s.push(pushData, DATA_OPR_PUSH_FUNC_NO, list)
	}

	idx := strings.LastIndex(pushData.Key, "/")
	if idx <= 0 {
		return
	}

	parentKey := pushData.Key[:idx]
	list, ok = s.cloneInfoObserverList(parentKey)
	if ok {
		s.push(pushData, DATA_OPR_PUSH_FUNC_NO, list)
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

func (s *Server) notifyConnChange(pushData *ConnChangePush) {
	list := s.cloneConnObserverList()
	s.push(pushData, CONN_CHANGE_FUNC_NO, list)
}

func (s *Server) push(pushData interface{}, funcNo uint16, list []*RegObserver) {
	if len(list) == 0 {
		return
	}

	packData, err := json.Marshal(pushData)
	if err != nil {
		s.logger.E("push json.Marshal err: ", err)
		return
	}

	h := rpc.NewPackHeader(PUSH_MARK, 0, funcNo)
	headerData, err := h.Marshal()
	if err != nil {
		s.logger.E("push PackHeader.Marshal err: ", err)
		return
	}

	payload := make([]rpc.ByteArray, 0, 2)
	payload = append(payload, headerData, packData)

	for _, observer := range list {
		s.WritePack(observer.SrvType, observer.SrvNo, payload...)
	}
}

func (s *Server) saveLoop() {
	for {
		// _, ok := <-s.evtSave.C
		// if !ok {
		// 	break
		// }

		err := s.evtSave.Wait()
		if err != nil {
			break
		}

		s.info.Save(s.savePath)
		if s.IsDebugMode() {
			s.info.Dump()
		}
	}
}
