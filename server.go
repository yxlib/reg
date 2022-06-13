// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reg

import (
	"encoding/json"
	"strings"

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
	*rpc.BaseServer
	info                   *RegInfo
	savePath               string
	mapKey2RegObserverList map[string]RegObserverList
	chanOprPush            chan *DataOprPush
	evtSave                *yx.Event
}

func NewServer(net rpc.Net, savePath string) *Server {
	s := &Server{
		BaseServer:             rpc.NewBaseServer(net),
		info:                   NewRegInfo(),
		savePath:               savePath,
		mapKey2RegObserverList: make(map[string]RegObserverList),
		chanOprPush:            make(chan *DataOprPush, MAX_PUSH_QUE),
		evtSave:                yx.NewEvent(),
	}

	s.SetMark(REG_MARK)
	s.SetInterceptor(&rpc.JsonInterceptor{})
	return s
}

func (s *Server) GetRegInfo() *RegInfo {
	return s.info
}

// func (s *Server) Save() {
// 	s.evtSave.Send()
// }

func (s *Server) UpdateSrv(srvType uint32, srvNo uint32, bTemp bool, dataBase64 string) {
	ok := s.info.HasSrv(srvType, srvNo)
	if !ok {
		s.info.AddSrv(srvType, srvNo, bTemp, dataBase64)
	} else {
		s.info.SetSrvData(srvType, srvNo, dataBase64)
	}

	s.evtSave.Send()

	key := GetSrvKey(srvType, srvNo)
	pushData := NewDataOprPush(key, DATA_OPR_TYPE_UPDATE)
	s.chanOprPush <- pushData
	// go s.notifyDataUpdate(key, DATA_OPR_TYPE_UPDATE)
}

func (s *Server) RemoveSrv(srvType uint32, srvNo uint32) {
	if s.info.HasSrv(srvType, srvNo) {
		s.info.RemoveSrv(srvType, srvNo)
		s.evtSave.Send()

		key := GetSrvKey(srvType, srvNo)
		pushData := NewDataOprPush(key, DATA_OPR_TYPE_REMOVE)
		s.chanOprPush <- pushData
		// go s.notifyDataUpdate(key, DATA_OPR_TYPE_REMOVE)
	}
}

func (s *Server) UpdateGlobalData(key string, dataBase64 string) {
	s.info.SetGlobalData(key, dataBase64)

	s.evtSave.Send()

	pushData := NewDataOprPush(key, DATA_OPR_TYPE_UPDATE)
	s.chanOprPush <- pushData
	// go s.notifyDataUpdate(reqData.Key, DATA_OPR_TYPE_UPDATE)
}

func (s *Server) RemoveGlobalData(key string) {
	if s.info.HasGlobalData(key) {
		s.info.RemoveGlobalData(key)
		s.evtSave.Send()

		pushData := NewDataOprPush(key, DATA_OPR_TYPE_REMOVE)
		s.chanOprPush <- pushData
		// go s.notifyDataUpdate(reqData.Key, DATA_OPR_TYPE_REMOVE)
	}
}

func (s *Server) Start() {
	go s.pushLoop()
	go s.saveLoop()
	s.BaseServer.Start()
}

func (s *Server) Stop() {
	s.BaseServer.Stop()
	s.evtSave.Close()
	close(s.chanOprPush)
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
	s.addObserver(key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopWatchSrv(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*StopWatchSrvReq)
	key := GetSrvKey(reqData.SrvType, reqData.SrvNo)
	s.removeObserver(key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnWatchSrvsByType(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*WatchSrvsByTypeReq)
	key := GetSrvTypeKey(reqData.SrvType)
	s.addObserver(key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopWatchSrvsByType(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*StopWatchSrvsByTypeReq)
	key := GetSrvTypeKey(reqData.SrvType)
	s.removeObserver(key, srcPeerType, srcPeerNo)

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
	s.addObserver(reqData.Key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopWatchGlobalData(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*StopWatchGlobalDataReq)
	s.removeObserver(reqData.Key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopAllWatch(req interface{}, resp interface{}, srcPeerType uint32, srcPeerNo uint32) error {
	reqData := req.(*StopAllWatchReq)
	s.removeAllObserverOfSrv(reqData.SrvType, reqData.SrvNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) addObserver(key string, srvType uint32, srvNo uint32) {
	o := &RegObserver{
		SrvType: srvType,
		SrvNo:   srvNo,
	}

	list, ok := s.mapKey2RegObserverList[key]
	if !ok {
		list = make([]*RegObserver, 0)
	} else if s.existObserver(list, srvType, srvNo) {
		return
	}

	s.mapKey2RegObserverList[key] = append(list, o)
}

func (s *Server) existObserver(list []*RegObserver, srvType uint32, srvNo uint32) bool {
	for _, observer := range list {
		if observer.IsSameObserver(srvType, srvNo) {
			return true
		}
	}

	return false
}

func (s *Server) removeObserver(key string, srvType uint32, srvNo uint32) {
	list, ok := s.mapKey2RegObserverList[key]
	if ok {
		s.mapKey2RegObserverList[key] = s.removeObserverFromList(list, srvType, srvNo)
	}
}

func (s *Server) removeAllObserverOfSrv(srvType uint32, srvNo uint32) {
	for key, list := range s.mapKey2RegObserverList {
		s.mapKey2RegObserverList[key] = s.removeObserverFromList(list, srvType, srvNo)
	}
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

func (s *Server) cloneObserverList(key string) (RegObserverList, bool) {
	list, ok := s.mapKey2RegObserverList[key]
	if !ok {
		return nil, false
	}

	cloneList := make(RegObserverList, len(list))
	copy(cloneList, list)
	return cloneList, ok
}

func (s *Server) pushLoop() {
	for {
		pushData, ok := <-s.chanOprPush
		if !ok {
			break
		}

		s.notifyDataUpdate(pushData.Key, pushData.Operate)
	}
}

func (s *Server) notifyDataUpdate(key string, opr int) {
	list, ok := s.cloneObserverList(key)
	if ok {
		s.pushDataUpdate(key, opr, list)
	}

	idx := strings.LastIndex(key, "/")
	if idx <= 0 {
		return
	}

	parentKey := key[:idx]
	list, ok = s.cloneObserverList(parentKey)
	if ok {
		s.pushDataUpdate(key, opr, list)
	}
}

func (s *Server) pushDataUpdate(key string, opr int, list []*RegObserver) {
	if len(list) == 0 {
		return
	}

	pushData := &DataOprPush{
		Key:     key,
		Operate: opr,
	}

	packData, err := json.Marshal(pushData)
	if err != nil {
		return
	}

	h := rpc.NewPackHeader([]byte(PUSH_MARK), 0, DATA_OPR_PUSH_FUNC_NO)
	headerData, err := h.Marshal()
	if err != nil {
		return
	}

	payload := make([]rpc.ByteArray, 0, 2)
	payload = append(payload, headerData, packData)

	for _, observer := range list {
		s.WritePack(payload, observer.SrvType, observer.SrvNo)
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
