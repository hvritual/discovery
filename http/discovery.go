package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Bilibili/discovery/errors"
	"github.com/Bilibili/discovery/model"
	gin "github.com/gin-gonic/gin"
	log "github.com/golang/glog"
)

const (
	_pollWaitSecond = 30 * time.Second
)

func register(c *gin.Context) {
	arg := new(model.ArgRegister)
	if err := c.Bind(arg); err != nil {
		return
	}
	i := model.NewInstance(arg)
	if i.Status == 0 || i.Status > 2 {
		result(c, nil, errors.ParamsErr)
		log.Errorf("register params status invalid")
		return
	}
	if arg.Metadata != "" {
		// check the metadata type is json
		if !json.Valid([]byte(arg.Metadata)) {
			result(c, nil, errors.ParamsErr)
			log.Errorf("register params() metadata(%v) invalid json", arg.Metadata)
			return
		}
	}
	// register replication
	if arg.DirtyTimestamp > 0 {
		i.DirtyTimestamp = arg.DirtyTimestamp
	}
	dis.Register(c, i, arg)
	result(c, nil, nil)
}

func renew(c *gin.Context) {
	arg := new(model.ArgRenew)
	if err := c.Bind(arg); err != nil {
		return
	}
	// renew
	instance, err := dis.Renew(c, arg)
	result(c, instance, err)
}

func cancel(c *gin.Context) {
	arg := new(model.ArgCancel)
	if err := c.Bind(arg); err != nil {
		result(c, nil, errors.ParamsErr)
		return
	}
	// cancel
	result(c, nil, dis.Cancel(c, arg))
}

func fetchAll(c *gin.Context) {
	result(c, dis.FetchAll(c), nil)
}

func fetch(c *gin.Context) {
	arg := new(model.ArgFetch)
	if err := c.Bind(arg); err != nil {
		result(c, nil, errors.ParamsErr)
		return
	}
	insInfo, err := dis.Fetch(c, arg)
	result(c, insInfo, err)
}

func fetchs(c *gin.Context) {
	arg := new(model.ArgFetchs)
	if err := c.Bind(arg); err != nil {
		return
	}
	ins, err := dis.Fetchs(c, arg)
	result(c, ins, err)
}

func poll(c *gin.Context) {
	arg := new(model.ArgPolls)
	if err := c.Bind(arg); err != nil {
		return
	}
	ch, _, err := dis.Polls(c, arg)
	if err != nil && err != errors.NotModified {
		result(c, nil, err)
		return
	}
	// wait for instance change
	select {
	case e := <-ch:
		result(c, resp{Data: e[arg.AppID[0]]}, nil)
		dis.PutChan(ch)
		// broadcast will delete all connections of appid
	case <-time.After(_pollWaitSecond):
		result(c, nil, errors.NotModified)
		dis.DelConns(arg)
	case <-c.Writer.(http.CloseNotifier).CloseNotify():
		result(c, nil, errors.NotModified)
		dis.DelConns(arg)
	}
}

func polls(c *gin.Context) {
	arg := new(model.ArgPolls)
	if err := c.Bind(arg); err != nil {
		return
	}
	if len(arg.AppID) != len(arg.LatestTimestamp) {
		result(c, nil, errors.ParamsErr)
		return
	}
	ch, new, err := dis.Polls(c, arg)
	if err != nil && err != errors.NotModified {
		result(c, nil, err)
		return
	}
	// wait for instance change
	select {
	case e := <-ch:
		result(c, e, nil)
		if new {
			dis.PutChan(ch)
		} else {
			dis.DelConns(arg)
		}
		// broadcast will delete all connections of appid
	case <-time.After(_pollWaitSecond):
		result(c, nil, errors.NotModified)
		dis.DelConns(arg)
	case <-c.Writer.(http.CloseNotifier).CloseNotify():
		result(c, nil, errors.NotModified)
		dis.DelConns(arg)
	}
}

func nodes(c *gin.Context) {
	result(c, dis.Nodes(c), nil)
}
