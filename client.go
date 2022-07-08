// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reg

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/yxlib/rpc"
	"github.com/yxlib/yx"
)

var (
	ErrRegCallFailed = errors.New("call failed")
)

type Client struct {
	rpcPeer  *rpc.Peer
	observer *Observer
	logger   *yx.Logger
	ec       *yx.ErrCatcher
}

func NewClient(rpcNet rpc.Net, observerNet rpc.Net, srvPeerType uint32, srvPeerNo uint32) *Client {
	return &Client{
		rpcPeer:  rpc.NewPeer(rpcNet, REG_MARK, srvPeerType, srvPeerNo),
		observer: NewObserver(observerNet, srvPeerType, srvPeerNo),
		logger:   yx.NewLogger("reg.Client"),
		ec:       yx.NewErrCatcher("reg.Client"),
	}
}

func (c *Client) Start() {
	c.rpcPeer.SetTimeout(TIME_OUT_SEC)

	go c.observer.Start()
	go c.rpcPeer.Start()
}

func (c *Client) Stop() {
	c.observer.Stop()
	c.rpcPeer.Stop()
}

func (c *Client) ListenDataOprPush(cb func(keyType int, key string, operate int)) {
	if cb == nil {
		return
	}

	for {
		pack, ok := c.observer.PopDataOprPack()
		if !ok {
			break
		}

		cb(pack.KeyType, pack.Key, pack.Operate)
	}
}

func (c *Client) ListenConnChangePush(cb func(srvType uint32, srvNo uint32, connChangeType int)) {
	if cb == nil {
		return
	}

	for {
		pack, ok := c.observer.PopConnChangePack()
		if !ok {
			break
		}

		cb(pack.SrvType, pack.SrvNo, pack.ConnChangeType)
	}
}

func (c *Client) FetchFuncList() error {
	err := c.rpcPeer.FetchFuncList(c.fetchRegFuncListCb)
	return c.ec.Throw("FetchFuncList", err)
}

func (c *Client) UpdateSrv(srvType uint32, srvNo uint32, bTemp bool, data []byte) error {
	req := &UpdateSrvReq{}
	req.SrvType = srvType
	req.SrvNo = srvNo
	req.IsTemp = bTemp
	req.DataBase64 = base64.StdEncoding.EncodeToString(data)

	resp := &BaseResp{}
	err := c.rpcCall("UpdateSrv", req, resp)
	return c.ec.Throw("UpdateSrv", err)
}

func (c *Client) RemoveSrv(srvType uint32, srvNo uint32) error {
	req := &RemoveSrvReq{
		SrvType: srvType,
		SrvNo:   srvNo,
	}

	resp := &BaseResp{}
	err := c.rpcCall("RemoveSrv", req, resp)
	return c.ec.Throw("RemoveSrv", err)
}

func (c *Client) GetSrv(srvType uint32, srvNo uint32) (*SrvInfo, error) {
	req := &GetSrvReq{
		SrvType: srvType,
		SrvNo:   srvNo,
	}

	resp := &GetSrvResp{}
	err := c.rpcCall("GetSrv", req, resp)
	if err != nil {
		return nil, c.ec.Throw("GetSrv", err)
	}

	return resp.Data, nil
}

func (c *Client) GetSrvByKey(key string) (*SrvInfo, error) {
	req := &GetSrvByKeyReq{
		Key: key,
	}

	resp := &GetSrvResp{}
	err := c.rpcCall("GetSrvByKey", req, resp)
	if err != nil {
		return nil, c.ec.Throw("GetSrvByKey", err)
	}

	return resp.Data, nil
}

func (c *Client) GetSrvsByType(srvType uint32) ([]*SrvInfo, error) {
	req := &GetSrvsByTypeReq{
		SrvType: srvType,
	}

	resp := &GetSrvsByTypeResp{}
	err := c.rpcCall("GetSrvsByType", req, resp)
	if err != nil {
		return nil, c.ec.Throw("GetSrvsByType", err)
	}

	return resp.Data, nil
}

func (c *Client) WatchSrv(srvType uint32, srvNo uint32) error {
	req := &WatchSrvReq{
		SrvType: srvType,
		SrvNo:   srvNo,
	}

	resp := &BaseResp{}
	err := c.rpcCall("WatchSrv", req, resp)
	return c.ec.Throw("WatchSrv", err)
}

func (c *Client) StopWatchSrv(srvType uint32, srvNo uint32) error {
	req := &StopWatchSrvReq{
		SrvType: srvType,
		SrvNo:   srvNo,
	}

	resp := &BaseResp{}
	err := c.rpcCall("StopWatchSrv", req, resp)
	return c.ec.Throw("StopWatchSrv", err)
}

func (c *Client) WatchSrvsByType(srvType uint32) error {
	req := &WatchSrvsByTypeReq{
		SrvType: srvType,
	}

	resp := &BaseResp{}
	err := c.rpcCall("WatchSrvsByType", req, resp)
	return c.ec.Throw("WatchSrvsByType", err)
}

func (c *Client) StopWatchSrvsByType(srvType uint32) error {
	req := &StopWatchSrvsByTypeReq{
		SrvType: srvType,
	}

	resp := &BaseResp{}
	err := c.rpcCall("StopWatchSrvsByType", req, resp)
	return c.ec.Throw("StopWatchSrvsByType", err)
}

func (c *Client) UpdateGlobalData(key string, data []byte) error {
	req := &UpdateGlobalDataReq{
		Key:        key,
		DataBase64: base64.StdEncoding.EncodeToString(data),
	}

	resp := &BaseResp{}
	err := c.rpcCall("UpdateGlobalData", req, resp)
	return c.ec.Throw("UpdateGlobalData", err)
}

func (c *Client) RemoveGlobalData(key string) error {
	req := &RemoveGlobalDataReq{
		Key: key,
	}

	resp := &BaseResp{}
	err := c.rpcCall("RemoveGlobalData", req, resp)
	return c.ec.Throw("RemoveGlobalData", err)
}

func (c *Client) GetGlobalData(key string) ([]byte, error) {
	req := &GetGlobalDataReq{
		Key: key,
	}

	resp := &GetGlobalDataResp{}
	err := c.rpcCall("GetGlobalData", req, resp)
	if err != nil {
		return nil, c.ec.Throw("GetGlobalData", err)
	}

	data, err := base64.StdEncoding.DecodeString(resp.DataBase64)
	if err != nil {
		return nil, c.ec.Throw("GetGlobalData", err)
	}

	return data, nil
}

func (c *Client) WatchGlobalData(key string) error {
	req := &WatchGlobalDataReq{
		Key: key,
	}

	resp := &BaseResp{}
	err := c.rpcCall("WatchGlobalData", req, resp)
	return c.ec.Throw("WatchGlobalData", err)
}

func (c *Client) StopWatchGlobalData(key string) error {
	req := &StopWatchGlobalDataReq{
		Key: key,
	}

	resp := &BaseResp{}
	err := c.rpcCall("StopWatchGlobalData", req, resp)
	return c.ec.Throw("StopWatchGlobalData", err)
}

func (c *Client) WatchConn() error {
	req := &WatchConnReq{}
	resp := &BaseResp{}
	err := c.rpcCall("WatchConn", req, resp)
	return c.ec.Throw("WatchConn", err)
}

func (c *Client) StopWatchConn() error {
	req := &StopWatchConnReq{}
	resp := &BaseResp{}
	err := c.rpcCall("StopWatchConn", req, resp)
	return c.ec.Throw("StopWatchConn", err)
}

func (c *Client) StopAllWatch(srvType uint32, srvNo uint32) error {
	req := &StopAllWatchReq{
		SrvType: srvType,
		SrvNo:   srvNo,
	}

	resp := &BaseResp{}
	err := c.rpcCall("StopAllWatch", req, resp)
	return c.ec.Throw("StopAllWatch", err)
}

func (c *Client) fetchRegFuncListCb(respData []byte) (*rpc.FetchFuncListResp, error) {
	resp := &rpc.FetchFuncListResp{}
	err := json.Unmarshal(respData, resp)
	if err != nil {
		c.logger.E("fetchRegFuncListCb json.Unmarshal err: ", err)
		return nil, c.ec.Throw("fetchRegFuncListCb", err)
	}

	return resp, nil
}

func (c *Client) rpcCall(funcName string, req interface{}, resp RegResp) error {
	var err error = nil
	defer c.ec.DeferThrow("rpcCall", &err)

	reqData, err := json.Marshal(req)
	if err != nil {
		c.logger.E("rpcCall json.Marshal err: ", err)
		return err
	}

	params := make([]rpc.ByteArray, 0)
	params = append(params, reqData)
	respData, err := c.rpcPeer.Call(funcName, params)
	if err != nil {
		c.logger.E("rpcCall rpcPeer.Call err: ", err)
		return err
	}

	err = json.Unmarshal(respData, resp)
	if err != nil {
		c.logger.E("rpcCall json.Unmarshal err: ", err)
		return err
	}

	if resp.GetResCode() != RES_CODE_SUCC {
		c.logger.W("rpcCall failed, ", resp.GetResCode(), ", ", resp.GetResMsg())
		err = ErrRegCallFailed
		return err
	}

	return nil
}
