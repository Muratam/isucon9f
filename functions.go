package main

import (
	crand "crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	// "sync"
)

func getSession(r *http.Request) *sessions.Session {
	session, _ := store.Get(r, sessionName)
	return session
}

func getUser(r *http.Request) (user User, errCode int, errMsg string) {
	session := getSession(r)
	userID, ok := session.Values["user_id"]
	if !ok {
		return user, http.StatusUnauthorized, "no session"
	}

	err := dbx.Get(&user, "SELECT * FROM `users` WHERE `id` = ?", userID)
	if err == sql.ErrNoRows {
		return user, http.StatusUnauthorized, "user not found"
	}
	if err != nil {
		log.Print(err)
		return user, http.StatusInternalServerError, "db error"
	}

	return user, http.StatusOK, ""
}

func secureRandomStr(b int) string {
	k := make([]byte, b)
	if _, err := crand.Read(k); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", k)
}

func distanceFareHandler(w http.ResponseWriter, r *http.Request) {

	distanceFareList := []DistanceFare{}

	query := "SELECT * FROM distance_fare_master"
	err := dbx.Select(&distanceFareList, query)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	for _, distanceFare := range distanceFareList {
		fmt.Fprintf(w, "%#v, %#v\n", distanceFare.Distance, distanceFare.Fare)
	}

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(w).Encode(distanceFareList)
}

func getDistanceFare(origToDestDistance float64) (int, error) {

	distanceFareList := []DistanceFare{}

	query := "SELECT distance,fare FROM distance_fare_master ORDER BY distance"
	err := dbx.Select(&distanceFareList, query)
	if err != nil {
		return 0, err
	}

	lastDistance := 0.0
	lastFare := 0
	for _, distanceFare := range distanceFareList {

		fmt.Println(origToDestDistance, distanceFare.Distance, distanceFare.Fare)
		if float64(lastDistance) < origToDestDistance && origToDestDistance < float64(distanceFare.Distance) {
			break
		}
		lastDistance = distanceFare.Distance
		lastFare = distanceFare.Fare
	}

	return lastFare, nil
}

func fareCalc(date time.Time, depStation int, destStation int, trainClass, seatClass string) (int, error) {
	//
	// 料金計算メモ
	// 距離運賃(円) * 期間倍率(繁忙期なら2倍等) * 車両クラス倍率(急行・各停等) * 座席クラス倍率(プレミアム・指定席・自由席)
	//
	var err error
	var fromStation, toStation Station

	query := "SELECT * FROM station_master WHERE id=?"

	// From
	err = dbx.Get(&fromStation, query, depStation)
	if err == sql.ErrNoRows {
		return 0, err
	}
	if err != nil {
		return 0, err
	}

	// To
	err = dbx.Get(&toStation, query, destStation)
	if err == sql.ErrNoRows {
		return 0, err
	}
	if err != nil {
		log.Print(err)
		return 0, err
	}

	fmt.Println("distance", math.Abs(toStation.Distance-fromStation.Distance))
	distFare, err := getDistanceFare(math.Abs(toStation.Distance - fromStation.Distance))
	if err != nil {
		return 0, err
	}
	fmt.Println("distFare", distFare)

	// 期間・車両・座席クラス倍率
	fareList := []Fare{}
	query = "SELECT * FROM fare_master WHERE train_class=? AND seat_class=? ORDER BY start_date"
	err = dbx.Select(&fareList, query, trainClass, seatClass)
	if err != nil {
		return 0, err
	}

	if len(fareList) == 0 {
		return 0, fmt.Errorf("fare_master does not exists")
	}

	selectedFare := fareList[0]
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	for _, fare := range fareList {
		if !date.Before(fare.StartDate) {
			fmt.Println(fare.StartDate, fare.FareMultiplier)
			selectedFare = fare
		}
	}

	fmt.Println("%%%%%%%%%%%%%%%%%%%")

	return int(float64(distFare) * selectedFare.FareMultiplier), nil
}

func makeReservationResponse(reservation Reservation) (ReservationResponse, error) {

	reservationResponse := ReservationResponse{}

	var departure, arrival string
	err := dbx.Get(
		&departure,
		"SELECT departure FROM train_timetable_master WHERE date=? AND train_class=? AND train_name=? AND station=?",
		reservation.Date.Format("2006/01/02"), reservation.TrainClass, reservation.TrainName, reservation.Departure,
	)
	if err != nil {
		return reservationResponse, err
	}
	err = dbx.Get(
		&arrival,
		"SELECT arrival FROM train_timetable_master WHERE date=? AND train_class=? AND train_name=? AND station=?",
		reservation.Date.Format("2006/01/02"), reservation.TrainClass, reservation.TrainName, reservation.Arrival,
	)
	if err != nil {
		return reservationResponse, err
	}

	reservationResponse.ReservationId = reservation.ReservationId
	reservationResponse.Date = reservation.Date.Format("2006/01/02")
	reservationResponse.Amount = reservation.Amount
	reservationResponse.Adult = reservation.Adult
	reservationResponse.Child = reservation.Child
	reservationResponse.Departure = reservation.Departure
	reservationResponse.Arrival = reservation.Arrival
	reservationResponse.TrainClass = reservation.TrainClass
	reservationResponse.TrainName = reservation.TrainName
	reservationResponse.DepartureTime = departure
	reservationResponse.ArrivalTime = arrival

	query := "SELECT * FROM seat_reservations WHERE reservation_id=?"
	err = dbx.Select(&reservationResponse.Seats, query, reservation.ReservationId)

	// 1つの予約内で車両番号は全席同じ
	reservationResponse.CarNumber = reservationResponse.Seats[0].CarNumber

	if reservationResponse.Seats[0].CarNumber == 0 {
		reservationResponse.SeatClass = "non-reserved"
	} else {
		// 座席種別を取得
		seat := Seat{}
		query = "SELECT * FROM seat_master WHERE train_class=? AND car_number=? AND seat_column=? AND seat_row=?"
		err = dbx.Get(
			&seat, query,
			reservation.TrainClass, reservationResponse.CarNumber,
			reservationResponse.Seats[0].SeatColumn, reservationResponse.Seats[0].SeatRow,
		)
		if err == sql.ErrNoRows {
			return reservationResponse, err
		}
		if err != nil {
			return reservationResponse, err
		}
		reservationResponse.SeatClass = seat.SeatClass
	}

	for i, v := range reservationResponse.Seats {
		// omit
		v.ReservationId = 0
		v.CarNumber = 0
		reservationResponse.Seats[i] = v
	}
	return reservationResponse, nil
}
