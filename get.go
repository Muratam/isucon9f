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
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(w).Encode(initialStationsByID)
}

func trainSearchHandler(w http.ResponseWriter, r *http.Request) {
	// 列車検索
	// GET /train/search?use_at=<ISO8601形式の時刻> & from=東京 & to=大阪
	// return 料金 空席情報 発駅と着駅の到着時刻

	jst := time.FixedZone("JST", 9*60*60)
	date, err := time.Parse(time.RFC3339, r.URL.Query().Get("use_at"))
	if err != nil {
		log.Print("failed to search train: failed to time.Parse:", err)
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

	if trainClass != "" {
		ok := false
		for _, cl := range usableTrainClassList {
			if cl == trainClass {
				ok = true
			}
		}
		if !ok {
			errorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		usableTrainClassList = []string{trainClass}
	}
	trainList := searchTrain(date, usableTrainClassList, isNobori)
	// 上りだったら駅リストを逆にする
	stations := []Station{}
	if isNobori {
		stations = initialStationByIDDesc
	} else {
		stations = initialStationsByID
	}
	type Deps struct {
		TrainName string `db:"train_name"`
		Station   string `db:"station"`
		Departure string `db:"departure"`
		Arrival   string `db:"arrival"`
	}
	depsList := []Deps{}
	arrsList := []Deps{}
	dbx.Select(
		&depsList,
		"SELECT departure, train_name,arrival,station FROM train_timetable_master WHERE date=? AND station = ?",
		date.Format("2006/01/02"),
		fromStation.Name)
	dbx.Select(
		&arrsList,
		"SELECT arrival, train_name,departure,station FROM train_timetable_master WHERE date=? AND station = ?",
		date.Format("2006/01/02"),
		toStation.Name)
	deps := map[string]string{}
	arrs := map[string]string{}
	for _, dep := range depsList {
		deps[dep.TrainName] = dep.Departure
		key := date.Format("2006/01/02") + ":" + dep.TrainName + dep.Station
		reservationCache.Store(key, ReservationCacheDepArr{dep.Departure, dep.Arrival})
	}
	for _, dep := range arrsList {
		arrs[dep.TrainName] = dep.Arrival
		key := date.Format("2006/01/02") + ":" + dep.TrainName + dep.Station
		reservationCache.Store(key, ReservationCacheDepArr{dep.Departure, dep.Arrival})
	}

	trainSearchResponseList := []TrainSearchResponse{}
	mayTrains := []Train{}
	for _, train := range trainList {
		isSeekedToFirstStation := false
		isContainsOriginStation := false
		isContainsDestStation := false
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
		}
		if !isContainsOriginStation || !isContainsDestStation {
			continue
		}
		departure := deps[train.TrainName]
		departureDate, err := time.Parse("2006/01/02 15:04:05 -07:00 MST", fmt.Sprintf("%s %s +09:00 JST", date.Format("2006/01/02"), departure))
		if err != nil {
			log.Print("failed to search train: failed to get departureDate:", err)
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !date.Before(departureDate) {
			// 乗りたい時刻より出発時刻が前なので除外
			continue
		}
		mayTrains = append(mayTrains, train)
		if len(mayTrains) >= 10 {
			break
		}
	}
	chunk := getAvailableSeatsChunk(mayTrains, fromStation, toStation)
	for _, train := range mayTrains {
		departure := deps[train.TrainName]
		arrival := arrs[train.TrainName]
		premium_avail_seats, premium_smoke_avail_seats, reserved_avail_seats, reserved_smoke_avail_seats := getAvailableSeatsCount(chunk, train.TrainClass, train.TrainName)
		// 料金計算
		premiumFare, err := fareCalc(date, fromStation.ID, toStation.ID, train.TrainClass, "premium")
		if err != nil {
			log.Print("failed to search train: failed to calc premium fare", err)
			errorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		premiumFare = premiumFare*adult + premiumFare/2*child
		reservedFare, err := fareCalc(date, fromStation.ID, toStation.ID, train.TrainClass, "reserved")
		if err != nil {
			log.Print("failed to search train: failed to calc reserved fare", err)
			errorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		reservedFare = reservedFare*adult + reservedFare/2*child
		nonReservedFare, err := fareCalc(date, fromStation.ID, toStation.ID, train.TrainClass, "non-reserved")
		if err != nil {
			log.Print("failed to search train: failed to calc non reserved fare", err)
			errorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		nonReservedFare = nonReservedFare*adult + nonReservedFare/2*child
		toMark := func(x int) string {
			if x == 0 {
				return "×"
			}
			if x < 10 {
				return "△"
			}
			return "○"
		}
		trainSearchResponseList = append(trainSearchResponseList,
			TrainSearchResponse{
				train.TrainClass, train.TrainName, train.StartStation, train.LastStation,
				fromStation.Name, toStation.Name, departure, arrival,
				map[string]string{
					"premium":        toMark(premium_avail_seats),
					"premium_smoke":  toMark(premium_smoke_avail_seats),
					"reserved":       toMark(reserved_avail_seats),
					"reserved_smoke": toMark(reserved_smoke_avail_seats),
					"non_reserved":   "○",
				},
				map[string]int{
					"premium":        premiumFare,
					"premium_smoke":  premiumFare,
					"reserved":       reservedFare,
					"reserved_smoke": reservedFare,
					"non_reserved":   nonReservedFare,
				},
			})
	}
	resp, err := json.Marshal(trainSearchResponseList)
	if err != nil {
		log.Print("failed to search train: failed to json.Marshal", err)
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
		log.Print("failed to get seats: failed to parse time", err)
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	date = date.In(jst)

	if !checkAvailableDate(date) {
		log.Print("failed to get seats: invalid date", err)
		errorResponse(w, http.StatusNotFound, "予約可能期間外です")
		return
	}

	trainClass := r.URL.Query().Get("train_class")
	trainName := r.URL.Query().Get("train_name")
	carNumber, _ := strconv.Atoi(r.URL.Query().Get("car_number"))
	fromName := r.URL.Query().Get("from")
	toName := r.URL.Query().Get("to")

	// 対象列車の取得
	train, err := getTrainWithClass(date, trainName, trainClass)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "列車が存在しません")
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

	type Resv struct {
		Departure  string `json:"departure" db:"departure"`
		Arrival    string `json:"arrival" db:"arrival"`
		CarNumber  int    `json:"car_number,omitempty" db:"car_number"`
		SeatRow    int    `json:"seat_row" db:"seat_row"`
		SeatColumn string `json:"seat_column" db:"seat_column"`
	}
	resvs := []Resv{}
	query := `
	SELECT departure,arrival,sm.car_number,sm.seat_row,sm.seat_column
	FROM seat_master as sm
	INNER JOIN seat_reservations sr ON sr.car_number = sm.car_number AND sr.seat_row = sm.seat_row AND sr.seat_column = sm.seat_column
	INNER JOIN reservations r ON r.reservation_id = sr.reservation_id
	WHERE sm.train_class=? AND sm.car_number=? AND date=? AND train_name=?
 	`
	err = dbx.Select(&resvs, query, trainClass, carNumber, date.Format("2006/01/02"), trainName)
	if err != nil {
		log.Print("failed to get seats: failed to get seat list:", err)
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	toKey := func(carNumber int, seatRow int, seatColumn string) string {
		return strconv.Itoa(carNumber) + seatColumn + strconv.Itoa(seatRow)
	}
	occupiedMap := map[string]bool{}
	for _, resv := range resvs {
		departureStation, _ := getStationByName[resv.Departure]
		arrivalStation, _ := getStationByName[resv.Arrival]
		if train.IsNobori {
			// 上り
			if toStation.ID < arrivalStation.ID && fromStation.ID <= arrivalStation.ID {
			} else if toStation.ID >= departureStation.ID && fromStation.ID > departureStation.ID {
			} else {
				occupiedMap[toKey(resv.CarNumber, resv.SeatRow, resv.SeatColumn)] = true
			}
		} else {
			if fromStation.ID < departureStation.ID && toStation.ID <= departureStation.ID {
			} else if fromStation.ID >= arrivalStation.ID && toStation.ID > arrivalStation.ID {
			} else {
				occupiedMap[toKey(resv.CarNumber, resv.SeatRow, resv.SeatColumn)] = true
			}
		}
	}

	seatList := []Seat{}
	query = "SELECT * FROM seat_master WHERE train_class=? AND car_number=? ORDER BY seat_row, seat_column"
	err = dbx.Select(&seatList, query, trainClass, carNumber)
	if err != nil {
		log.Print("failed to get seats: failed to get seat list:", err)
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	seatInformationList := []SeatInformation{}
	for _, seat := range seatList {
		s := SeatInformation{seat.SeatRow, seat.SeatColumn, seat.SeatClass, seat.IsSmokingSeat, false}
		s.IsOccupied = occupiedMap[toKey(carNumber, seat.SeatRow, seat.SeatColumn)]
		seatInformationList = append(seatInformationList, s)
	}
	// 各号車の情報
	c := CarInformation{date.Format("2006/01/02"), trainClass, trainName, carNumber, seatInformationList, initialSimpleCarInformation[trainClassNameToIndex(trainClass)]}
	resp, err := json.Marshal(c)
	if err != nil {
		log.Print("failed to get seats: failed to json.Marshal", err)
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
		log.Print("failed to get user reservation: failed to get reservation list", err)
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
		log.Print("failed to get user reservation response: failed to strconv.ParseInt", err)
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
		log.Print("failed to get user reservation response: failed to make reservation response", err)
		errorResponse(w, http.StatusBadRequest, err.Error())
		log.Println("makeReservationResponse() ", err)
		return
	}

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(w).Encode(reservationResponse)
}
