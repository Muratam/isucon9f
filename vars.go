package main

import (
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"time"
)

const (
	sessionName   = "session_isutrain"
	availableDays = 100
	cancelInterval = 4 * time.Second
)

var (
	store sessions.Store = sessions.NewCookieStore([]byte(secureRandomStr(20)))
)

var (
	banner        = `ISUTRAIN API`
	TrainClassMap = map[string]string{"express": "最速", "semi_express": "中間", "local": "遅いやつ"}
)

var dbx *sqlx.DB
