package main

import (
	crand "crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
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
	if !idToUserServer.Get(strconv.Itoa(int(userID.(int64))), &user) {
		return user, http.StatusUnauthorized, "user not found"
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
		log.Print(err)
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
	lastDistance := 0.0
	lastFare := 0

	for _, distanceFare := range initialDistanceFares {
		// fmt.Println(origToDestDistance, distanceFare.Distance, distanceFare.Fare)
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
	if depStation < 1 || depStation > len(initialStationsByID) {
		return 0, sql.ErrNoRows
	}
	if destStation < 1 || destStation > len(initialStationsByID) {
		return 0, sql.ErrNoRows
	}
	fromStation := getStationByID(depStation)
	toStation := getStationByID(destStation)
	// fmt.Println("distance", math.Abs(toStation.Distance-fromStation.Distance))
	distFare, err := getDistanceFare(math.Abs(toStation.Distance - fromStation.Distance))
	if err != nil {
		return 0, err
	}
	// fmt.Println("distFare", distFare)

	// 期間・車両・座席クラス倍率
	fareList, ok := FaresfromtrainClassSeatClass[trainClass+seatClass]
	if !ok {
		return 0, err
	}
	selectedFare := fareList[0]
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	for _, fare := range fareList {
		if !date.Before(fare.StartDate) {
			// fmt.Println(fare.StartDate, fare.FareMultiplier)
			selectedFare = fare
		}
	}
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
		seat := initialSimpleCarInformation[trainClassNameToIndex(reservation.TrainClass)][reservationResponse.CarNumber-1]
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

func checkAvailableDate(date time.Time) bool {
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	t := time.Date(2020, 1, 1, 0, 0, 0, 0, jst)
	t = t.AddDate(0, 0, availableDays)

	return date.Before(t)
}

func getUsableTrainClassList(fromStation Station, toStation Station) []string {
	usable := map[string]string{}

	for key, value := range TrainClassMap {
		usable[key] = value
	}

	if !fromStation.IsStopExpress {
		delete(usable, "express")
	}
	if !fromStation.IsStopSemiExpress {
		delete(usable, "semi_express")
	}
	if !fromStation.IsStopLocal {
		delete(usable, "local")
	}

	if !toStation.IsStopExpress {
		delete(usable, "express")
	}
	if !toStation.IsStopSemiExpress {
		delete(usable, "semi_express")
	}
	if !toStation.IsStopLocal {
		delete(usable, "local")
	}

	ret := []string{}
	for _, v := range usable {
		ret = append(ret, v)
	}

	return ret
}

var availableSeatMapss = func() [][]map[int]bool {
	result := make([][]map[int]bool, 3)
	for ri, trainClass := range fromTrainClassI {
		// 全ての座席を取得する
		premium_avail_seats := getSeatsWithIsSmoking(false, "premium", trainClass)
		premium_smoke_avail_seats := getSeatsWithIsSmoking(true, "premium", trainClass)
		reserved_avail_seats := getSeatsWithIsSmoking(false, "reserved", trainClass)
		reserved_smoke_avail_seats := getSeatsWithIsSmoking(true, "reserved", trainClass)
		availableSeatMaps := make([]map[int]bool, 4)
		for i := 0; i < 4; i++ {
			availableSeatMaps[i] = map[int]bool{}
		}
		for _, seat := range premium_avail_seats {
			availableSeatMaps[0][seat.CarNumber*1000+seat.SeatRow*10+SeatClassNameToIndex(seat.SeatColumn)] = true
		}
		for _, seat := range premium_smoke_avail_seats {
			availableSeatMaps[1][seat.CarNumber*1000+seat.SeatRow*10+SeatClassNameToIndex(seat.SeatColumn)] = true
		}
		for _, seat := range reserved_avail_seats {
			availableSeatMaps[2][seat.CarNumber*1000+seat.SeatRow*10+SeatClassNameToIndex(seat.SeatColumn)] = true
		}
		for _, seat := range reserved_smoke_avail_seats {
			availableSeatMaps[3][seat.CarNumber*1000+seat.SeatRow*10+SeatClassNameToIndex(seat.SeatColumn)] = true
		}
		result[ri] = availableSeatMaps
	}
	return result
}()

func (train Train) getAvailableSeatsCount(fromStation Station, toStation Station) (int, int, int, int, error) {
	// 指定種別の空き座席を返す
	var err error

	// すでに取られている予約を取得する
	query := `
	SELECT sr.reservation_id, sr.car_number, sr.seat_row, sr.seat_column
	FROM seat_reservations sr, reservations r, seat_master s, station_master std, station_master sta
	WHERE
		r.train_name=? AND
		r.train_class=? AND
		r.reservation_id=sr.reservation_id AND
		s.train_class=r.train_class AND
		s.car_number=sr.car_number AND
		s.seat_column=sr.seat_column AND
		s.seat_row=sr.seat_row AND
		std.name=r.departure AND
		sta.name=r.arrival
	`
	if train.IsNobori {
		query += "AND ((sta.id < ? AND ? <= std.id) OR (sta.id < ? AND ? <= std.id) OR (? < sta.id AND std.id < ?))"
	} else {
		query += "AND ((std.id <= ? AND ? < sta.id) OR (std.id <= ? AND ? < sta.id) OR (sta.id < ? AND ? < std.id))"
	}

	seatReservationList := []SeatReservation{}
	err = dbx.Select(&seatReservationList, query, train.TrainName, train.TrainClass, fromStation.ID, fromStation.ID, toStation.ID, toStation.ID, fromStation.ID, toStation.ID)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	availableSeatMaps := availableSeatMapss[trainClassNameToIndex(train.TrainClass)]
	embeds := make([]map[int]bool, 4)
	for _, seatReservation := range seatReservationList {
		key := seatReservation.CarNumber*1000 + seatReservation.SeatRow*10 + SeatClassNameToIndex(seatReservation.SeatColumn)
		for i := 0; i < 4; i++ {
			if availableSeatMaps[i][key] {
				embeds[i][key] = true
			}
		}
	}
	return len(availableSeatMaps[0]) - len(embeds[0]),
		len(availableSeatMaps[1]) - len(embeds[1]),
		len(availableSeatMaps[2]) - len(embeds[2]),
		len(availableSeatMaps[3]) - len(embeds[3]), nil
}
