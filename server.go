// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reg

import (
	"errors"

	"github.com/yxlib/rpc"
	"github.com/yxlib/server"
	"github.com/yxlib/yx"
)

var (
	ErrSrvServNotExist       = errors.New("server not exists")
	ErrSrvServTypeNotExist   = errors.New("server type not exists")
	ErrSrvGlobalDataNotExist = errors.New("global data not exists")
)

type Service struct {
	*server.BaseService

	logger *yx.Logger
	ec     *yx.ErrCatcher
}

func NewService() *Service {
	return &Service{
		BaseService: server.NewBaseService(REG_SRV),
		logger:      yx.NewLogger("reg.Server"),
		ec:          yx.NewErrCatcher("reg.Server"),
	}

	// s.Start()

	// s.SetName(REG_SRV)
	// s.SetInterceptor(&rpc.JsonInterceptor{})
	// return s
}

// func (s *Service) GetRegInfo() *RegInfo {
// 	return s.info
// }

// func (s *Server) Save() {
// 	s.evtSave.Send()
// }

func (s *Service) OnUpdateSrv(req *server.Request, resp *server.Response) (int32, error) {
	reqData, _ := req.ExtData.(*UpdateSrvReq)
	RegCenter.UpdateSrv(reqData.SrvType, reqData.SrvNo, reqData.IsTemp, reqData.DataBase64)

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnRemoveSrv(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*RemoveSrvReq)
	RegCenter.RemoveSrv(reqData.SrvType, reqData.SrvNo)

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnGetSrv(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*GetSrvReq)
	respData := resp.ExtData.(*GetSrvResp)

	regInfo := RegCenter.GetRegInfo()
	srvInfo, ok := regInfo.GetSrvInfo(reqData.SrvType, reqData.SrvNo)
	if !ok {
		return RES_CODE_SRV_NOT_EXISTS, s.ec.Throw("OnGetSrv", ErrSrvServNotExist)
	}
	// if ok {
	// 	respData.SetResult(RES_CODE_SUCC, "")
	// } else {
	// 	respData.SetResult(RES_CODE_SRV_NOT_EXISTS, "server not exists")
	// }

	respData.Data = srvInfo
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnGetSrvByKey(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*GetSrvByKeyReq)
	respData := resp.ExtData.(*GetSrvByKeyResp)

	regInfo := RegCenter.GetRegInfo()
	srvInfo, ok := regInfo.GetSrvInfoByKey(reqData.Key)
	if !ok {
		return RES_CODE_SRV_NOT_EXISTS, s.ec.Throw("OnGetSrvByKey", ErrSrvServNotExist)
	}
	// if ok {
	// 	respData.SetResult(RES_CODE_SUCC, "")
	// } else {
	// 	respData.SetResult(RES_CODE_SRV_NOT_EXISTS, "server not exists")
	// }

	respData.Data = srvInfo
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnGetSrvsByType(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*GetSrvsByTypeReq)
	respData := resp.ExtData.(*GetSrvsByTypeResp)

	regInfo := RegCenter.GetRegInfo()
	infos, ok := regInfo.GetAllSrvInfos(reqData.SrvType)
	if !ok {
		return RES_CODE_SRV_TYPE_NOT_EXISTS, s.ec.Throw("OnGetSrvsByType", ErrSrvServTypeNotExist)
	}
	// if ok {
	// 	respData.SetResult(RES_CODE_SUCC, "")
	// } else {
	// 	respData.SetResult(RES_CODE_SRV_TYPE_NOT_EXISTS, "server type not exists")
	// }

	respData.Data = infos
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnWatchSrv(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*WatchSrvReq)
	key := GetSrvKey(reqData.SrvType, reqData.SrvNo)
	RegCenter.AddInfoObserver(key, uint32(req.Src.PeerType), uint32(req.Src.PeerNo))

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnStopWatchSrv(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*StopWatchSrvReq)
	key := GetSrvKey(reqData.SrvType, reqData.SrvNo)
	RegCenter.RemoveInfoObserver(key, uint32(req.Src.PeerType), uint32(req.Src.PeerNo))

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnWatchSrvsByType(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*WatchSrvsByTypeReq)
	key := GetSrvTypeKey(reqData.SrvType)
	RegCenter.AddInfoObserver(key, uint32(req.Src.PeerType), uint32(req.Src.PeerNo))

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnStopWatchSrvsByType(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*StopWatchSrvsByTypeReq)
	key := GetSrvTypeKey(reqData.SrvType)
	RegCenter.RemoveInfoObserver(key, uint32(req.Src.PeerType), uint32(req.Src.PeerNo))

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnUpdateGlobalData(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*UpdateGlobalDataReq)
	RegCenter.UpdateGlobalData(reqData.Key, reqData.DataBase64)

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnRemoveGlobalData(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*RemoveGlobalDataReq)
	RegCenter.RemoveGlobalData(reqData.Key)

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnGetGlobalData(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*GetGlobalDataReq)
	respData := resp.ExtData.(*GetGlobalDataResp)

	regInfo := RegCenter.GetRegInfo()
	dataBase64, ok := regInfo.GetGlobalData(reqData.Key)
	if !ok {
		return RES_CODE_GLOBAL_DATA_NOT_EXISTS, s.ec.Throw("OnGetGlobalData", ErrSrvGlobalDataNotExist)
	}
	// if ok {
	// 	respData.SetResult(RES_CODE_SUCC, "")
	// } else {
	// 	respData.SetResult(RES_CODE_GLOBAL_DATA_NOT_EXISTS, "global data not exists")
	// }

	respData.DataBase64 = dataBase64
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnWatchGlobalData(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*WatchGlobalDataReq)
	RegCenter.AddInfoObserver(reqData.Key, uint32(req.Src.PeerType), uint32(req.Src.PeerNo))

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnStopWatchGlobalData(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*StopWatchGlobalDataReq)
	RegCenter.RemoveInfoObserver(reqData.Key, uint32(req.Src.PeerType), uint32(req.Src.PeerNo))

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnWatchConn(req *server.Request, resp *server.Response) (int32, error) {
	// reqData := req.(*WatchConnReq)
	RegCenter.AddConnObserver(uint32(req.Src.PeerType), uint32(req.Src.PeerNo))

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnStopWatchConn(req *server.Request, resp *server.Response) (int32, error) {
	// reqData := req.(*StopWatchConnReq)
	RegCenter.RemoveConnObserver(uint32(req.Src.PeerType), uint32(req.Src.PeerNo))

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}

func (s *Service) OnStopAllWatch(req *server.Request, resp *server.Response) (int32, error) {
	reqData := req.ExtData.(*StopAllWatchReq)
	RegCenter.RemoveAllObserverOfSrv(reqData.SrvType, reqData.SrvNo)

	// respData := resp.(*BaseResp)
	// respData.SetResult(RES_CODE_SUCC, "")
	return rpc.RES_CODE_SUCC, nil
}
