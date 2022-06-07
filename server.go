// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reg

import (
	"encoding/json"
	"strings"

	"github.com/yxlib/rpc"
)

const (
	DATA_OPR_TYPE_UPDATE = iota + 1
	DATA_OPR_TYPE_REMOVE
)

type RegObserver struct {
	SrvType uint16
	SrvNo   uint16
}

func NewRegObserver(srvType uint16, srvNo uint16) *RegObserver {
	return &RegObserver{
		SrvType: srvType,
		SrvNo:   srvNo,
	}
}

func (o *RegObserver) IsSameObserver(srvType uint16, srvNo uint16) bool {
	return (o.SrvType == srvType && o.SrvNo == srvNo)
}

type RegObserverList = []*RegObserver

type Server struct {
	*rpc.BaseServer
	info                   *RegInfo
	mapKey2RegObserverList map[string]RegObserverList
}

func NewServer(net rpc.Net) *Server {
	s := &Server{
		BaseServer:             rpc.NewBaseServer(net),
		info:                   NewRegInfo(),
		mapKey2RegObserverList: make(map[string]RegObserverList),
	}

	s.SetMark(REG_MARK)
	s.SetInterceptor(&rpc.JsonInterceptor{})
	return s
}

func (s *Server) OnUpdateSrv(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
	reqData := req.(*UpdateSrvReq)
	ok := s.info.HasSrv(reqData.SrvType, reqData.SrvNo)
	if !ok {
		s.info.AddSrv(reqData.SrvType, reqData.SrvNo, reqData.IsTemp, reqData.DataBase64)
	} else {
		s.info.SetSrvData(reqData.SrvType, reqData.SrvNo, reqData.DataBase64)
	}

	key := s.info.GetSrvKey(reqData.SrvType, reqData.SrvNo)
	go s.notifyDataUpdate(key, DATA_OPR_TYPE_UPDATE)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnRemoveSrv(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
	reqData := req.(*RemoveSrvReq)
	if s.info.HasSrv(reqData.SrvType, reqData.SrvNo) {
		s.info.RemoveSrv(reqData.SrvType, reqData.SrvNo)

		key := s.info.GetSrvKey(reqData.SrvType, reqData.SrvNo)
		go s.notifyDataUpdate(key, DATA_OPR_TYPE_REMOVE)
	}

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnGetSrv(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
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

func (s *Server) OnGetSrvsByType(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
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

func (s *Server) OnWatchSrv(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
	reqData := req.(*WatchSrvReq)
	key := s.info.GetSrvKey(reqData.SrvType, reqData.SrvNo)
	s.addObserver(key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopWatchSrv(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
	reqData := req.(*StopWatchSrvReq)
	key := s.info.GetSrvKey(reqData.SrvType, reqData.SrvNo)
	s.removeObserver(key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnWatchSrvsByType(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
	reqData := req.(*WatchSrvsByTypeReq)
	key := s.info.GetSrvTypeKey(reqData.SrvType)
	s.addObserver(key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopWatchSrvsByType(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
	reqData := req.(*StopWatchSrvsByTypeReq)
	key := s.info.GetSrvTypeKey(reqData.SrvType)
	s.removeObserver(key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnUpdateGlobalData(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
	reqData := req.(*UpdateGlobalDataReq)
	s.info.SetGlobalData(reqData.Key, reqData.DataBase64)

	go s.notifyDataUpdate(reqData.Key, DATA_OPR_TYPE_UPDATE)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnRemoveGlobalData(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
	reqData := req.(*RemoveGlobalDataReq)

	if s.info.HasGlobalData(reqData.Key) {
		s.info.RemoveGlobalData(reqData.Key)
		go s.notifyDataUpdate(reqData.Key, DATA_OPR_TYPE_REMOVE)
	}

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnGetGlobalData(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
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

func (s *Server) OnWatchGlobalData(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
	reqData := req.(*WatchGlobalDataReq)
	s.addObserver(reqData.Key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopWatchGlobalData(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
	reqData := req.(*StopWatchGlobalDataReq)
	s.removeObserver(reqData.Key, srcPeerType, srcPeerNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) OnStopAllWatch(req interface{}, resp interface{}, srcPeerType uint16, srcPeerNo uint16) error {
	reqData := req.(*StopAllWatchReq)
	s.removeAllObserverOfSrv(reqData.SrvType, reqData.SrvNo)

	respData := resp.(*BaseResp)
	respData.SetResult(RES_CODE_SUCC, "")
	return nil
}

func (s *Server) addObserver(key string, srvType uint16, srvNo uint16) {
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

func (s *Server) existObserver(list []*RegObserver, srvType uint16, srvNo uint16) bool {
	for _, observer := range list {
		if observer.IsSameObserver(srvType, srvNo) {
			return true
		}
	}

	return false
}

func (s *Server) removeObserver(key string, srvType uint16, srvNo uint16) {
	list, ok := s.mapKey2RegObserverList[key]
	if ok {
		s.mapKey2RegObserverList[key] = s.removeObserverFromList(list, srvType, srvNo)
	}
}

func (s *Server) removeAllObserverOfSrv(srvType uint16, srvNo uint16) {
	for key, list := range s.mapKey2RegObserverList {
		s.mapKey2RegObserverList[key] = s.removeObserverFromList(list, srvType, srvNo)
	}
}

func (s *Server) removeObserverFromList(list []*RegObserver, srvType uint16, srvNo uint16) []*RegObserver {
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
