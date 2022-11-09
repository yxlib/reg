// Copyright 2022 Guan Jianchang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reg

import (
	"encoding/json"

	"github.com/yxlib/rpc"
	"github.com/yxlib/yx"
)

type Observer struct {
	net                rpc.Net
	chanDataOprPush    chan *DataOprPush
	chanConnChangePush chan *ConnChangePush
	logger             *yx.Logger
}

func NewObserver(net rpc.Net, peerType uint32, peerNo uint32) *Observer {
	o := &Observer{
		net:                net,
		chanDataOprPush:    make(chan *DataOprPush),
		chanConnChangePush: make(chan *ConnChangePush),
		logger:             yx.NewLogger("reg.Observer"),
	}

	o.net.SetMark(PUSH_MARK, peerType, peerNo)
	return o
}

func (o *Observer) Start() {
	o.readPackLoop()
}

func (o *Observer) Stop() {
	o.net.Close()
}

func (o *Observer) PopDataOprPack() (*DataOprPush, bool) {
	pack, ok := <-o.chanDataOprPush
	return pack, ok
}

func (o *Observer) PopConnChangePack() (*ConnChangePush, bool) {
	pack, ok := <-o.chanConnChangePush
	return pack, ok
}

func (o *Observer) readPackLoop() {
	for {
		data, err := o.net.ReadRpcPack()
		if err != nil {
			break
		}

		h := rpc.NewPackHeader(PUSH_MARK, 0, 0)
		err = h.Unmarshal(data.Payload)
		if err != nil {
			continue
		}

		headerLen := h.GetHeaderLen()
		o.handlePack(h.FuncNo, data.Payload[headerLen:])
	}

	close(o.chanDataOprPush)
	close(o.chanConnChangePush)
}

func (o *Observer) handlePack(funcNo uint16, payload []byte) {
	if funcNo == DATA_OPR_PUSH_FUNC_NO {
		pushPack := &DataOprPush{}
		err := json.Unmarshal(payload, pushPack)
		if err != nil {
			o.logger.E("handlePack json.Unmarshal err: ", err)
			return
		}

		o.chanDataOprPush <- pushPack

	} else if funcNo == CONN_CHANGE_FUNC_NO {
		pushPack := &ConnChangePush{}
		err := json.Unmarshal(payload, pushPack)
		if err != nil {
			o.logger.E("handlePack json.Unmarshal err: ", err)
			return
		}

		o.chanConnChangePush <- pushPack
	}
}
