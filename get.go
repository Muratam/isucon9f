package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"goji.io/pat"
)

func settingsHandler(w http.ResponseWriter, r *http.Request) {
	payment_api := os.Getenv("PAYMENT_API")
	if payment_api == "" {
		payment_api = "http://localhost:5000"
	}

	settings := Settings{
		PaymentAPI: payment_api,
	}

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(w).Encode(settings)
}

func getStationsHandler(w http.ResponseWriter, r *http.Request) {
	/*
		駅一覧
			GET /api/stations

		return []Station{}
	*/

	stations := []Station{}

	query := "SELECT * FROM station_master ORDER BY id"
	err := dbx.Select(&stations, query)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(w).Encode(stations)
}

func trainSearchHandler(w http.ResponseWriter, r *http.Request) {
	// 列車検索
	// GET /train/search?use_at=<ISO8601形式の時刻> & from=東京 & to=大阪
	// return 料金 空席情報 発駅と着駅の到着時刻

	jst := time.FixedZone("JST", 9*60*60)
	date, err := time.Parse(time.RFC3339, r.URL.Query().Get("use_at"))
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	date = date.In(jst)

	if !checkAvailableDate(date) {
		errorResponse(w, http.StatusNotFound, "予約可能期間外です")
		return
	}

	trainClass := r.URL.Query().Get("train_class")
	fromName := r.URL.Query().Get("from")
	toName := r.URL.Query().Get("to")

	adult, _ := strconv.Atoi(r.URL.Query().Get("adult"))
	child, _ := strconv.Atoi(r.URL.Query().Get("child"))

	var fromStation, toStation Station
	fromStation, ok := getStationByName[fromName]
	if !ok {
		log.Print("fromStation: no rows")
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	toStation, ok = getStationByName[toName]
	if !ok {
		log.Print("toStation: no rows")
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	isNobori := fromStation.Distance > toStation.Distance
	usableTrainClassList := getUsableTrainClassList(fromStation, toStation)

	var inQuery string
	var inArgs []interface{}

	if trainClass == "" {
		query := "SELECT * FROM train_master WHERE date=? AND train_class IN (?) AND is_nobori=?"
		inQuery, inArgs, err = sqlx.In(query, date.Format("2006/01/02"), usableTrainClassList, isNobori)
	} else {
		query := "SELECT * FROM train_master WHERE date=? AND train_class IN (?) AND is_nobori=? AND train_class=?"
		inQuery, inArgs, err = sqlx.In(query, date.Format("2006/01/02"), usableTrainClassList, isNobori, trainClass)
	}
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	trainList := []Train{}
	err = dbx.Select(&trainList, inQuery, inArgs...)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// 上りだったら駅リストを逆にする
	stations := []Station{}
	if isNobori {
		stations = initialStationByIDDesc
	} else {
		stations = initialStationsByID
	}
	// fmt.Println("From", fromStation)
	// fmt.Println("To", toStation)

	trainSearchResponseList := []TrainSearchResponse{}

	for _, train := range trainList {
		isSeekedToFirstStation := false
		isContainsOriginStation := false
		isContainsDestStation := false
		i := 0

		for _, station := range stations {

			// 駅リストを列車の発駅まで読み飛ばして頭出しをする
			// 列車の発駅以前は止まらないので無視して良い
			if !isSeekedToFirstStation {
				if station.Name == train.StartStation {
					isSeekedToFirstStation = true
				} else {
					continue
				}
			}
			// 発駅を経路中に持つ編成の場合フラグを立てる
			if station.ID == fromStation.ID {
				isContainsOriginStation = true
			}
			if station.ID == toStation.ID {
				if isContainsOriginStation {
					// 発駅と着駅を経路中に持つ編成の場合
					isContainsDestStation = true
					break
				} else {
					// 出発駅より先に終点が見つかったとき
					fmt.Println("なんかおかしい")
					break
				}
			}
			if station.Name == train.LastStation {
				// 駅が見つからないまま当該編成の終点に着いてしまったとき
				break
			}
			i++
		}

		if isContainsOriginStation && isContainsDestStation {
			// 列車情報

			// 所要時間
			var departure, arrival string

			err = dbx.Get(&departure, "SELECT departure FROM train_timetable_master WHERE date=? AND train_class=? AND train_name=? AND station=?", date.Format("2006/01/02"), train.TrainClass, train.TrainName, fromStation.Name)
			if err != nil {
				errorResponse(w, http.StatusInternalServerError, err.Error())
				return
			}

			departureDate, err := time.Parse("2006/01/02 15:04:05 -07:00 MST", fmt.Sprintf("%s %s +09:00 JST", date.Format("2006/01/02"), departure))
			if err != nil {
				errorResponse(w, http.StatusInternalServerError, err.Error())
				return
			}

			if !date.Before(departureDate) {
				// 乗りたい時刻より出発時刻が前なので除外
				continue
			}

			err = dbx.Get(&arrival, "SELECT arrival FROM train_timetable_master WHERE date=? AND train_class=? AND train_name=? AND station=?", date.Format("2006/01/02"), train.TrainClass, train.TrainName, toStation.Name)
			if err != nil {
				errorResponse(w, http.StatusInternalServerError, err.Error())
				return
			}

			premium_avail_seats, premium_smoke_avail_seats, reserved_avail_seats, reserved_smoke_avail_seats, err := train.getAvailableSeatsCount(fromStation, toStation)
			if err != nil {
				errorResponse(w, http.StatusBadRequest, err.Error())
				return
			}

			premium_avail := "○"
			if premium_avail_seats == 0 {
				premium_avail = "×"
			} else if premium_avail_seats < 10 {
				premium_avail = "△"
			}

			premium_smoke_avail := "○"
			if premium_smoke_avail_seats == 0 {
				premium_smoke_avail = "×"
			} else if premium_smoke_avail_seats < 10 {
				premium_smoke_avail = "△"
			}

			reserved_avail := "○"
			if reserved_avail_seats == 0 {
				reserved_avail = "×"
			} else if reserved_avail_seats < 10 {
				reserved_avail = "△"
			}

			reserved_smoke_avail := "○"
			if reserved_smoke_avail_seats == 0 {
				reserved_smoke_avail = "×"
			} else if reserved_smoke_avail_seats < 10 {
				reserved_smoke_avail = "△"
			}

			// 空席情報
			seatAvailability := map[string]string{
				"premium":        premium_avail,
				"premium_smoke":  premium_smoke_avail,
				"reserved":       reserved_avail,
				"reserved_smoke": reserved_smoke_avail,
				"non_reserved":   "○",
			}

			// 料金計算
			premiumFare, err := fareCalc(date, fromStation.ID, toStation.ID, train.TrainClass, "premium")
			if err != nil {
				errorResponse(w, http.StatusBadRequest, err.Error())
				return
			}
			premiumFare = premiumFare*adult + premiumFare/2*child

			reservedFare, err := fareCalc(date, fromStation.ID, toStation.ID, train.TrainClass, "reserved")
			if err != nil {
				errorResponse(w, http.StatusBadRequest, err.Error())
				return
			}
			reservedFare = reservedFare*adult + reservedFare/2*child

			nonReservedFare, err := fareCalc(date, fromStation.ID, toStation.ID, train.TrainClass, "non-reserved")
			if err != nil {
				errorResponse(w, http.StatusBadRequest, err.Error())
				return
			}
			nonReservedFare = nonReservedFare*adult + nonReservedFare/2*child

			fareInformation := map[string]int{
				"premium":        premiumFare,
				"premium_smoke":  premiumFare,
				"reserved":       reservedFare,
				"reserved_smoke": reservedFare,
				"non_reserved":   nonReservedFare,
			}

			trainSearchResponseList = append(trainSearchResponseList, TrainSearchResponse{
				train.TrainClass, train.TrainName, train.StartStation, train.LastStation,
				fromStation.Name, toStation.Name, departure, arrival, seatAvailability, fareInformation,
			})

			if len(trainSearchResponseList) >= 10 {
				break
			}
		}
	}
	resp, err := json.Marshal(trainSearchResponseList)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Write(resp)

}

func trainSeatsHandler(w http.ResponseWriter, r *http.Request) {

	//	指定した列車の座席列挙
	// GET /train/seats?date=2020-03-01&train_class=のぞみ&train_name=96号&car_number=2&from=大阪&to=東京

	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	date, err := time.Parse(time.RFC3339, r.URL.Query().Get("date"))
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	date = date.In(jst)

	if !checkAvailableDate(date) {
		errorResponse(w, http.StatusNotFound, "予約可能期間外です")
		return
	}

	trainClass := r.URL.Query().Get("train_class")
	trainName := r.URL.Query().Get("train_name")
	carNumber, _ := strconv.Atoi(r.URL.Query().Get("car_number"))
	fromName := r.URL.Query().Get("from")
	toName := r.URL.Query().Get("to")

	// 対象列車の取得
	var train Train
	query := "SELECT * FROM train_master WHERE date=? AND train_class=? AND train_name=?"
	err = dbx.Get(&train, query, date.Format("2006/01/02"), trainClass, trainName)
	if err == sql.ErrNoRows {
		errorResponse(w, http.StatusNotFound, "列車が存在しません")
	}
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	// From
	fromStation, ok := getStationByName[fromName]
	if !ok {
		log.Print("fromStation: no rows")
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	// To
	toStation, ok := getStationByName[toName]

	if !ok {
		log.Print("toStation: no rows")
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	usableTrainClassList := getUsableTrainClassList(fromStation, toStation)
	usable := false
	for _, v := range usableTrainClassList {
		if v == train.TrainClass {
			usable = true
		}
	}
	if !usable {
		err = fmt.Errorf("invalid train_class")
		log.Print(err)
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	seatList := []Seat{}

	query = "SELECT * FROM seat_master WHERE train_class=? AND car_number=? ORDER BY seat_row, seat_column"
	err = dbx.Select(&seatList, query, trainClass, carNumber)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var seatInformationList []SeatInformation

	for _, seat := range seatList {

		s := SeatInformation{seat.SeatRow, seat.SeatColumn, seat.SeatClass, seat.IsSmokingSeat, false}

		reservationList := []Reservation{}

		query := `SELECT r.* FROM seat_reservations s, reservations r WHERE r.reservation_id=s.reservation_id AND r.date=? AND r.train_class=? AND r.train_name=? AND car_number=? AND seat_row=? AND seat_column=?`

		err = dbx.Select(
			&reservationList, query,
			date.Format("2006/01/02"),
			seat.TrainClass,
			trainName,
			seat.CarNumber,
			seat.SeatRow,
			seat.SeatColumn,
		)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		for _, reservation := range reservationList {
			departureStation, _ := getStationByName[reservation.Departure]
			arrivalStation, _ := getStationByName[reservation.Arrival]
			if train.IsNobori {
				// 上り
				if toStation.ID < arrivalStation.ID && fromStation.ID <= arrivalStation.ID {
					// pass
				} else if toStation.ID >= departureStation.ID && fromStation.ID > departureStation.ID {
					// pass
				} else {
					s.IsOccupied = true
				}

			} else {
				// 下り

				if fromStation.ID < departureStation.ID && toStation.ID <= departureStation.ID {
					// pass
				} else if fromStation.ID >= arrivalStation.ID && toStation.ID > arrivalStation.ID {
					// pass
				} else {
					s.IsOccupied = true
				}

			}
		}

		fmt.Println(s.IsOccupied)
		seatInformationList = append(seatInformationList, s)
	}

	// 各号車の情報

	simpleCarInformationList := []SimpleCarInformation{}
	seat := Seat{}
	query = "SELECT * FROM seat_master WHERE train_class=? AND car_number=? ORDER BY seat_row, seat_column LIMIT 1"
	i := 1
	for {
		err = dbx.Get(&seat, query, trainClass, i)
		if err != nil {
			break
		}
		simpleCarInformationList = append(simpleCarInformationList, SimpleCarInformation{i, seat.SeatClass})
		i = i + 1
	}

	c := CarInformation{date.Format("2006/01/02"), trainClass, trainName, carNumber, seatInformationList, simpleCarInformationList}
	resp, err := json.Marshal(c)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Write(resp)
}

func getAuthHandler(w http.ResponseWriter, r *http.Request) {

	// userID取得
	user, errCode, errMsg := getUser(r)
	if errCode != http.StatusOK {
		errorResponse(w, errCode, errMsg)
		log.Printf("%s", errMsg)
		return
	}

	resp := AuthResponse{user.Email}
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(w).Encode(resp)
}

func userReservationsHandler(w http.ResponseWriter, r *http.Request) {
	/*
		ログイン
		POST /auth/login
	*/
	user, errCode, errMsg := getUser(r)
	if errCode != http.StatusOK {
		errorResponse(w, errCode, errMsg)
		return
	}
	reservationList := []Reservation{}

	query := "SELECT * FROM reservations WHERE user_id=?"
	err := dbx.Select(&reservationList, query, user.ID)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	reservationResponseList := []ReservationResponse{}

	for _, r := range reservationList {
		res, err := makeReservationResponse(r)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, err.Error())
			log.Println("makeReservationResponse()", err)
			return
		}
		reservationResponseList = append(reservationResponseList, res)
	}

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(w).Encode(reservationResponseList)
}
func userReservationResponseHandler(w http.ResponseWriter, r *http.Request) {
	/*
		ログイン
		POST /auth/login
	*/
	user, errCode, errMsg := getUser(r)
	if errCode != http.StatusOK {
		errorResponse(w, errCode, errMsg)
		return
	}
	itemIDStr := pat.Param(r, "item_id")
	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil || itemID <= 0 {
		errorResponse(w, http.StatusBadRequest, "incorrect item id")
		return
	}

	reservation := Reservation{}
	query := "SELECT * FROM reservations WHERE reservation_id=? AND user_id=?"
	err = dbx.Get(&reservation, query, itemID, user.ID)
	if err == sql.ErrNoRows {
		errorResponse(w, http.StatusNotFound, "Reservation not found")
		return
	}
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	reservationResponse, err := makeReservationResponse(reservation)

	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		log.Println("makeReservationResponse() ", err)
		return
	}

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(w).Encode(reservationResponse)
}
