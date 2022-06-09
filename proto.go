// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reg

const (
	REG_MARK              = "REG_SRV"
	TIME_OUT_SEC          = 3
	PUSH_MARK             = "REG_PUSH"
	DATA_OPR_PUSH_FUNC_NO = 1
)

const (
	RES_CODE_SUCC                   = 0
	RES_CODE_SRV_NOT_EXISTS         = 100
	RES_CODE_SRV_TYPE_NOT_EXISTS    = 101
	RES_CODE_GLOBAL_DATA_NOT_EXISTS = 102
)

// RegResp
type RegResp interface {
	SetResult(code int, msg string)
	GetResCode() int
	GetResMsg() string
}

type BaseResp struct {
	ResCode int    `json:"resCode"`
	Msg     string `json:"msg"`
}

func (r *BaseResp) SetResult(code int, msg string) {
	r.ResCode = code
	r.Msg = msg
}

func (r *BaseResp) GetResCode() int {
	return r.ResCode
}

func (r *BaseResp) GetResMsg() string {
	return r.Msg
}

// UpdateSrv
type UpdateSrvReq struct {
	SrvInfo
}

// type UpdateSrvResp struct {
// 	BaseResp
// }

// RemoveSrv
type RemoveSrvReq struct {
	SrvType uint32 `json:"type"`
	SrvNo   uint32 `json:"no"`
}

// type RemoveSrvResp struct {
// 	BaseResp
// }

// GetSrv
type GetSrvReq struct {
	SrvType uint32 `json:"type"`
	SrvNo   uint32 `json:"no"`
}

type GetSrvResp struct {
	BaseResp
	Data *SrvInfo `json:"data"`
}

// GetSrvByKey
type GetSrvByKeyReq struct {
	Key string `json:"key"`
}

type GetSrvByKeyResp struct {
	BaseResp
	Data *SrvInfo `json:"data"`
}

// GetSrvsByType
type GetSrvsByTypeReq struct {
	SrvType uint32 `json:"type"`
}

type GetSrvsByTypeResp struct {
	BaseResp
	Data []*SrvInfo `json:"data"`
}

// WatchSrv
type WatchSrvReq struct {
	SrvType uint32 `json:"type"`
	SrvNo   uint32 `json:"no"`
}

// type WatchSrvResp struct {
// 	BaseResp
// }

// StopWatchSrv
type StopWatchSrvReq struct {
	SrvType uint32 `json:"type"`
	SrvNo   uint32 `json:"no"`
}

// type StopWatchSrvResp struct {
// 	BaseResp
// }

// WatchSrvsByType
type WatchSrvsByTypeReq struct {
	SrvType uint32 `json:"type"`
}

// type WatchSrvsByTypeResp struct {
// 	BaseResp
// }

// StopWatchSrvsByType
type StopWatchSrvsByTypeReq struct {
	SrvType uint32 `json:"type"`
}

// type StopWatchSrvsByTypeResp struct {
// 	BaseResp
// }

// UpdateGlobalData
type UpdateGlobalDataReq struct {
	Key        string `json:"key"`
	DataBase64 string `json:"data"`
}

// type UpdateGlobalDataResp struct {
// 	BaseResp
// }

// RemoveGlobalData
type RemoveGlobalDataReq struct {
	Key string `json:"key"`
}

// type RemoveGlobalDataResp struct {
// 	BaseResp
// }

// GetGlobalData
type GetGlobalDataReq struct {
	Key string `json:"key"`
}

type GetGlobalDataResp struct {
	BaseResp
	DataBase64 string `json:"data"`
}

// WatchGlobalData
type WatchGlobalDataReq struct {
	Key string `json:"key"`
}

// type WatchGlobalDataResp struct {
// 	BaseResp
// }

// StopWatchGlobalData
type StopWatchGlobalDataReq struct {
	Key string `json:"key"`
}

// type StopWatchGlobalDataResp struct {
// 	BaseResp
// }

type StopAllWatchReq struct {
	SrvType uint32 `json:"type"`
	SrvNo   uint32 `json:"no"`
}

// type StopAllWatchResp struct {
// 	BaseResp
// }

type DataOprPush struct {
	Key     string `json:"key"`
	Operate int    `json:"opr"`
}

func NewDataOprPush(key string, operate int) *DataOprPush {
	return &DataOprPush{
		Key:     key,
		Operate: operate,
	}
}
