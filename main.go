package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	goji "goji.io"
	"goji.io/pat"
	// "sync"
)

// DB定義

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World")
}
func dummyHandler(w http.ResponseWriter, r *http.Request) {
	messageResponse(w, "ok")
}
func messageResponse(w http.ResponseWriter, message string) {
	e := map[string]interface{}{
		"is_error": false,
		"message":  message,
	}
	errResp, _ := json.Marshal(e)
	w.Write(errResp)
}

func errorResponse(w http.ResponseWriter, errCode int, message string) {
	e := map[string]interface{}{
		"is_error": true,
		"message":  message,
	}
	errResp, _ := json.Marshal(e)

	w.WriteHeader(errCode)
	w.Write(errResp)
}

func main() {
	go func() { log.Println(http.ListenAndServe(":9876", nil)) }()
	// MySQL関連のお膳立て
	var err error

	host := os.Getenv("MYSQL_HOSTNAME")
	if host == "" {
		host = "127.0.0.1"
	}
	// port := os.Getenv("MYSQL_PORT")
	// if port == "" {
	// 	port = "3306"
	// }
	port := "13306"
	_, err = strconv.Atoi(port)
	if err != nil {
		port = "3306"
	}
	user := os.Getenv("MYSQL_USER")
	if user == "" {
		user = "isutrain"
	}
	dbname := os.Getenv("MYSQL_DATABASE")
	if dbname == "" {
		dbname = "isutrain"
	}
	password := os.Getenv("MYSQL_PASSWORD")
	if password == "" {
		password = "isutrain"
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local&interpolateParams=true",
		user,
		password,
		host,
		port,
		dbname,
	)

	dbx, err = sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to connect to DB: %s.", err.Error())
	}
	defer dbx.Close()

	// HTTP
	mux := goji.NewMux()
	mux.HandleFunc(pat.Post("/initialize"), initializeHandler)
	mux.HandleFunc(pat.Get("/api/settings"), settingsHandler)
	// 予約関係
	mux.HandleFunc(pat.Get("/api/stations"), getStationsHandler)
	mux.HandleFunc(pat.Get("/api/train/search"), trainSearchHandler)
	mux.HandleFunc(pat.Get("/api/train/seats"), trainSeatsHandler)
	mux.HandleFunc(pat.Post("/api/train/reserve"), trainReservationHandler)
	mux.HandleFunc(pat.Post("/api/train/reservation/commit"), reservationPaymentHandler)
	// 認証関連
	mux.HandleFunc(pat.Get("/api/auth"), getAuthHandler)
	mux.HandleFunc(pat.Post("/api/auth/signup"), signUpHandler)
	mux.HandleFunc(pat.Post("/api/auth/login"), loginHandler)
	mux.HandleFunc(pat.Post("/api/auth/logout"), logoutHandler)
	mux.HandleFunc(pat.Get("/api/user/reservations"), userReservationsHandler)
	mux.HandleFunc(pat.Get("/api/user/reservations/:item_id"), userReservationResponseHandler)
	mux.HandleFunc(pat.Post("/api/user/reservations/:item_id/cancel"), userReservationCancelHandler)

	fmt.Println(banner)
	err = http.ListenAndServe(":8000", mux)

	log.Fatal(err)
}
