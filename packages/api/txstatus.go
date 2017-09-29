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

package api

import (
	"fmt"
	"net/http"

	"github.com/EGaaS/go-egaas-mvp/packages/consts"
	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	logger "github.com/EGaaS/go-egaas-mvp/packages/log"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
)

type txstatusResult struct {
	BlockID string `json:"blockid"`
	Message string `json:"errmsg"`
}

func txstatus(w http.ResponseWriter, r *http.Request, data *apiData) error {
	logger.LogDebug(consts.FuncStarted, "")
	var status txstatusResult
	ts := &model.TransactionStatus{}
	binTx := converter.HexToBin(data.params["hash"])
	notFound, err := ts.Get(binTx)
	if notFound {
		logger.LogError(consts.RouteError, fmt.Sprintf("can't find transaction status by hash %s", data.params[`hash`].(string)))
		return errorAPI(w, `hash has not been found`, http.StatusBadRequest)
	}
	if err != nil {
		return errorAPI(w, err.Error(), http.StatusInternalServerError)
	}
	if ts.BlockID > 0 {
		status.BlockID = converter.Int64ToStr(ts.BlockID)
	}
	status.Message = ts.Error
	data.result = &status
	return nil
}
