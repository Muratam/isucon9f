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

	"sync"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
)

func getSession(r *http.Request) *sessions.Session {
	session, _ := store.Get(r, sessionName)
	return session
}

var sessionCache = sync.Map{}

func sessUserID(r *http.Request) (int64, bool) {
	cookie, err := r.Cookie(sessionName)
	if err == nil {
		if val, ok := sessionCache.Load(cookie.Value); ok {
			return val.(int64), true
		} else {
			session := getSession(r)
			userID, ok := session.Values["user_id"]
			if !ok {
				return -1, false
			}
			sessionCache.Store(cookie.Value, userID.(int64))
			return userID.(int64), true
		}
	}
	return -1, false
}

func getUser(r *http.Request) (user User, errCode int, errMsg string) {
	userID, ok := sessUserID(r)
	if !ok {
		return user, http.StatusUnauthorized, "no session"
	}
	if !idToUserServer.Get(strconv.Itoa(int(userID)), &user) {
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
		"SELECT departure FROM train_timetable_master WHERE date=? AND train_name=? AND station=?",
		reservation.Date.Format("2006/01/02"), reservation.TrainName, reservation.Departure,
	)
	if err != nil {
		return reservationResponse, err
	}
	err = dbx.Get(
		&arrival,
		"SELECT arrival FROM train_timetable_master WHERE date=? AND train_name=? AND station=?",
		reservation.Date.Format("2006/01/02"), reservation.TrainName, reservation.Arrival,
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

// train_name+train_class 毎に返却
func getAvailableSeatsChunk(trains []Train, fromStation Station, toStation Station) map[string][]int {
	// 指定種別の空き座席を返す
	type Resv struct {
		TrainClass string `json:"train_class" db:"train_class"`
		TrainName  string `json:"train_name" db:"train_name"`
		Departure  string `json:"departure" db:"departure"`
		Arrival    string `json:"arrival" db:"arrival"`
		CarNumber  int    `json:"car_number,omitempty" db:"car_number"`
		SeatRow    int    `json:"seat_row" db:"seat_row"`
		SeatColumn string `json:"seat_column" db:"seat_column"`
	}
	allowed := make([]string, len(trains))
	for i, train := range trains {
		allowed[i] = train.TrainName
	}
	// 3倍ありそう
	resvs := []Resv{}
	query := `
	SELECT train_class,train_name,departure,arrival,car_number,seat_row,seat_column
	FROM reservations r
	INNER JOIN seat_reservations s
		ON r.reservation_id = s.reservation_id
	WHERE r.train_name IN ?
	`
	dbx.Select(&resvs, query, allowed)
	result := map[string][]int{}
	for _, resv := range resvs {
		xx, _ := strconv.Atoi(resv.TrainName)
		isNobori := xx%2 == 1
		departureStation, _ := getStationByName[resv.Departure]
		arrivalStation, _ := getStationByName[resv.Arrival]
		if isNobori { // 上り
			if toStation.ID < arrivalStation.ID && fromStation.ID <= arrivalStation.ID {
				continue
			} else if toStation.ID >= departureStation.ID && fromStation.ID > departureStation.ID {
				continue
			}
		} else { // 下り
			if fromStation.ID < departureStation.ID && toStation.ID <= departureStation.ID {
				continue
			} else if fromStation.ID >= arrivalStation.ID && toStation.ID > arrivalStation.ID {
				continue
			}
		}
		key := resv.CarNumber*1000 + resv.SeatRow*10 + SeatClassNameToIndex(resv.SeatColumn)
		trkey := resv.TrainName + resv.TrainClass
		if pre, ok := result[trkey]; ok {
			result[trkey] = append(pre, key)
		} else {
			result[trkey] = []int{key}
		}
	}
	return result
}
func getAvailableSeatsCount(chunk map[string][]int, trainClass string, trainName string) (int, int, int, int) {
	availableSeatMaps := availableSeatMapss[trainClassNameToIndex(trainClass)]
	embeds := make([]map[int]bool, 4)
	for i := 0; i < 4; i++ {
		embeds[i] = map[int]bool{}
	}
	trkey := trainName + trainClass
	if _, ok := chunk[trkey]; ok {
		for _, key := range chunk[trkey] {
			for i := 0; i < 4; i++ {
				if availableSeatMaps[i][key] {
					embeds[i][key] = true
				}
			}
		}
	}
	return len(availableSeatMaps[0]) - len(embeds[0]),
		len(availableSeatMaps[1]) - len(embeds[1]),
		len(availableSeatMaps[2]) - len(embeds[2]),
		len(availableSeatMaps[3]) - len(embeds[3])
}
