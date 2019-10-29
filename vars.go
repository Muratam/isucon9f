package main

import (
	"time"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

const (
	sessionName    = "session_isutrain"
	availableDays  = 200
	cancelInterval = 100 * time.Millisecond
)

var (
	store sessions.Store = sessions.NewCookieStore([]byte(secureRandomStr(20)))
)

var (
	banner        = `ISUTRAIN API`
	TrainClassMap = map[string]string{"express": "最速", "semi_express": "中間", "local": "遅いやつ"}
)

var dbx *sqlx.DB

// TODO: 複数台だとたぶんこれはだめなのでいいかんじに
// var idToUserServer = NewSyncMapServerConn(GetMasterServerAddress()+":8884", isMasterServerIP)
var idToUserServer = NewSyncMapServerConn("127.0.0.1:8884", true)
