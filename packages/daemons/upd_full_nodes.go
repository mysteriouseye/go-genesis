// Copyright 2016 The go-daylight Authors
// This file is part of the go-daylight library.
//
// The go-daylight library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-daylight library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-daylight library. If not, see <http://www.gnu.org/licenses/>.

package daemons

import (
	"context"
	"fmt"
	"time"

	"encoding/hex"
	"github.com/EGaaS/go-egaas-mvp/packages/consts"

	"github.com/EGaaS/go-egaas-mvp/packages/config/syspar"
	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/crypto"
	logger "github.com/EGaaS/go-egaas-mvp/packages/log"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/parser"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
	"github.com/EGaaS/go-egaas-mvp/packages/utils/tx"
	"gopkg.in/vmihailenco/msgpack.v2"
)

// UpdFullNodes sends UpdFullNodes transactions
func UpdFullNodes(d *daemon, ctx context.Context) error {
	logger.LogError(consts.FuncStarted, "")
	d.sleepTime = 60 * time.Second

	locked, err := DbLock(ctx, d.goRoutineName)
	if !locked || err != nil {
		logger.LogError(consts.DBError, err)
		return err
	}
	defer DbUnlock(d.goRoutineName)

	infoBlock := &model.InfoBlock{}
	err = infoBlock.GetInfoBlock()
	if err != nil {
		logger.LogError(consts.DBError, err)
		return err
	}

	if infoBlock.BlockID == 0 {
		d.sleepTime = 10 * time.Second
		return nil
	}

	nodeConfig := &model.Config{}
	err = nodeConfig.GetConfig()
	if err != nil {
		logger.LogError(consts.DBError, err)
		return err

	}
	myStateID := nodeConfig.StateID
	myWalletID := nodeConfig.DltWalletID
	logger.LogDebug(consts.DebugMessage, myWalletID)
	// If we are in the list of those who are able to generate the blocks
	fullNode := &model.FullNode{}
	err = fullNode.FindNode(myStateID, myWalletID, myStateID, myWalletID)
	if err != nil {
		logger.LogError(consts.DBError, err)
		return err
	}

	fullNodeID := fullNode.ID
	logger.LogDebug(consts.DebugMessage, fmt.Sprintf("fullNodeID = %d", fullNodeID))
	if fullNodeID == 0 {
		d.sleepTime = 10 * time.Second // because 1s is too small for non-full nodes
		return nil
	}

	curTime := time.Now().Unix()

	// check if the time of the last updating passed
	updFn := &model.UpdFullNode{}
	err = updFn.Read()
	if err != nil {
		logger.LogError(consts.DBError, err)
		return err
	}

	updFullNodes := int64(updFn.Time)
	if curTime-updFullNodes <= syspar.GetUpdFullNodesPeriod() {
		logger.LogDebug(consts.DebugMessage, "curTime-adminTime <= consts.UPD_FULL_NODES_PERIOD")
		return nil
	}

	myNodeKey := &model.MyNodeKey{}
	err = myNodeKey.GetNodeWithMaxBlockID()
	if err != nil {
		logger.LogError(consts.DBError, err)
		return err
	}

	tr := tx.UpdFullNodes{
		Header: tx.Header{
			Type:      int(utils.TypeInt("UpdFullNodes")),
			Time:      curTime,
			UserID:    myWalletID,
			StateID:   0,
			PublicKey: myNodeKey.PublicKey,
		},
	}

	binSign, err := crypto.Sign(hex.EncodeToString(myNodeKey.PrivateKey), tr.ForSign())
	if err != nil {
		logger.LogError(consts.CryptoError, err)
		return err
	}
	tr.Header.BinSignatures = binSign

	data, err := msgpack.Marshal(tr)
	if err != nil {
		return err
	}
	data = append(converter.DecToBin(int64(tr.Type), 1), data...)

	hash, err := crypto.Hash(data)
	if err != nil {
		logger.LogError(consts.CryptoError, err)
		return err
	}

	queueTx := &model.QueueTx{Hash: hash}
	err = queueTx.DeleteTx()
	if err != nil {
		logger.LogError(consts.DBError, err)
		return err
	}

	queueTx.Data = data
	queueTx.Hash = hash
	err = queueTx.Save()
	if err != nil {
		logger.LogError(consts.DBError, err)
		return nil
	}

	p := new(parser.Parser)
	err = p.TxParser(hash, data, true)
	if err != nil {
		logger.LogError(consts.ParserError, err)
		return err
	}
	return nil
}
