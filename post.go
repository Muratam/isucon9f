package main

import (
	"bytes"
	crand "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"goji.io/pat"
	"golang.org/x/crypto/pbkdf2"
)

func initializeHandler(w http.ResponseWriter, r *http.Request) {
	/*
		initialize
	*/

	dbx.Exec("TRUNCATE seat_reservations")
	dbx.Exec("TRUNCATE reservations")
	dbx.Exec("TRUNCATE users")

	resp := InitializeResponse{
		availableDays,
		"golang",
	}
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(w).Encode(resp)
}

func trainReservationHandler(w http.ResponseWriter, r *http.Request) {
	/*
		列車の席予約API　支払いはまだ
		POST /api/train/reserve
			{
				"date": "2020-12-31T07:57:00+09:00",
				"train_name": "183",
				"train_class": "中間",
				"car_number": 7,
				"is_smoking_seat": false,
				"seat_class": "reserved",
				"departure": "東京",
				"arrival": "名古屋",
				"child": 2,
				"adult": 1,
				"column": "A",
				"seats": [
					{
					"row": 3,
					"column": "B"
					},
						{
					"row": 4,
					"column": "C"
					}
				]
		}
		レスポンスで予約IDを返す
		reservationResponse(w http.ResponseWriter, errCode int, id int, ok bool, message string)
	*/

	// json parse
	req := new(TrainReservationRequest)
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "JSON parseに失敗しました")
		log.Println(err.Error())
		return
	}

	// 乗車日の日付表記統一
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	date, err := time.Parse(time.RFC3339, req.Date)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "時刻のparseに失敗しました")
		log.Println(err.Error())
	}
	date = date.In(jst)

	if !checkAvailableDate(date) {
		errorResponse(w, http.StatusNotFound, "予約可能期間外です")
		return
	}

	fromStation, ok := getStationByName[req.Departure]
	if !ok {
		errorResponse(w, http.StatusNotFound, fmt.Sprintf("乗車駅データがみつかりません %s", req.Departure))
		log.Println(err.Error())
		return
	}
	toStation, ok := getStationByName[req.Arrival]
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "乗車駅データの取得に失敗しました")
		log.Println(err.Error())
		return
	}

	// 列車データを取得
	tmas := Train{}
	query := "SELECT * FROM train_master WHERE date=? AND train_class=? AND train_name=?"
	err = dbx.Get(
		&tmas, query,
		date.Format("2006/01/02"),
		req.TrainClass,
		req.TrainName,
	)
	if err == sql.ErrNoRows {
		errorResponse(w, http.StatusNotFound, "列車データがみつかりません")
		log.Println(err.Error())
		return
	}
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "列車データの取得に失敗しました")
		log.Println(err.Error())
		return
	}

	// Departure
	departureStation, ok := getStationByName[tmas.StartStation]
	if !ok {
		errorResponse(w, http.StatusNotFound, "リクエストされた列車の始発駅データがみつかりません")
		log.Println(err.Error())
		return
	}
	// Arrive
	arrivalStation, ok := getStationByName[tmas.LastStation]
	if !ok {
		errorResponse(w, http.StatusNotFound, "リクエストされた列車の終着駅データがみつかりません")
		log.Println(err.Error())
		return
	}

	switch req.TrainClass {
	case "最速":
		if !fromStation.IsStopExpress || !toStation.IsStopExpress {
			errorResponse(w, http.StatusBadRequest, "最速の止まらない駅です")
			return
		}
	case "中間":
		if !fromStation.IsStopSemiExpress || !toStation.IsStopSemiExpress {
			errorResponse(w, http.StatusBadRequest, "中間の止まらない駅です")
			return
		}
	case "遅いやつ":
		if !fromStation.IsStopLocal || !toStation.IsStopLocal {
			errorResponse(w, http.StatusBadRequest, "遅いやつの止まらない駅です")
			return
		}
	default:
		errorResponse(w, http.StatusBadRequest, "リクエストされた列車クラスが不明です")
		log.Println(err.Error())
		return
	}

	// 運行していない区間を予約していないかチェックする
	if tmas.IsNobori {
		if fromStation.ID > departureStation.ID || toStation.ID > departureStation.ID {
			errorResponse(w, http.StatusBadRequest, "リクエストされた区間に列車が運行していない区間が含まれています")
			return
		}
		if arrivalStation.ID >= fromStation.ID || arrivalStation.ID > toStation.ID {
			errorResponse(w, http.StatusBadRequest, "リクエストされた区間に列車が運行していない区間が含まれています")
			return
		}
	} else {
		if fromStation.ID < departureStation.ID || toStation.ID < departureStation.ID {
			errorResponse(w, http.StatusBadRequest, "リクエストされた区間に列車が運行していない区間が含まれています")
			return
		}
		if arrivalStation.ID <= fromStation.ID || arrivalStation.ID < toStation.ID {
			errorResponse(w, http.StatusBadRequest, "リクエストされた区間に列車が運行していない区間が含まれています")
			return
		}
	}

	//	あいまい座席検索
	//	seatsが空白の時に発動する
	switch len(req.Seats) {
	case 0:
		if req.SeatClass == "non-reserved" {
			break // non-reservedはそもそもあいまい検索もせずダミーのRow/Columnで予約を確定させる。
		}
		//当該列車・号車中の空き座席検索
		var train Train
		query := "SELECT * FROM train_master WHERE date=? AND train_class=? AND train_name=?"
		err = dbx.Get(&train, query, date.Format("2006/01/02"), req.TrainClass, req.TrainName)
		if err == sql.ErrNoRows {
			panic(err)
		}
		if err != nil {
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

		req.Seats = []RequestSeat{} // 座席リクエスト情報は空に
		for carnum := 1; carnum <= 16; carnum++ {
			seatList := []Seat{}
			query = "SELECT * FROM seat_master WHERE train_class=? AND car_number=? AND seat_class=? AND is_smoking_seat=? ORDER BY seat_row, seat_column"
			err = dbx.Select(&seatList, query, req.TrainClass, carnum, req.SeatClass, req.IsSmokingSeat)
			if err != nil {
				errorResponse(w, http.StatusBadRequest, err.Error())
				return
			}

			var seatInformationList []SeatInformation
			for _, seat := range seatList {
				s := SeatInformation{seat.SeatRow, seat.SeatColumn, seat.SeatClass, seat.IsSmokingSeat, false}
				seatReservationList := []SeatReservation{}
				query = "SELECT s.* FROM seat_reservations s, reservations r WHERE r.date=? AND r.train_class=? AND r.train_name=? AND car_number=? AND seat_row=? AND seat_column=? FOR UPDATE"
				err = dbx.Select(
					&seatReservationList, query,
					date.Format("2006/01/02"),
					seat.TrainClass,
					req.TrainName,
					seat.CarNumber,
					seat.SeatRow,
					seat.SeatColumn,
				)
				if err != nil {
					errorResponse(w, http.StatusBadRequest, err.Error())
					return
				}

				for _, seatReservation := range seatReservationList {
					reservation := Reservation{}
					query = "SELECT * FROM reservations WHERE reservation_id=? FOR UPDATE"
					err = dbx.Get(&reservation, query, seatReservation.ReservationId)
					if err != nil {
						panic(err)
					}

					var departureStation, arrivalStation Station
					query = "SELECT * FROM station_master WHERE name=?"

					err = dbx.Get(&departureStation, query, reservation.Departure)
					if err != nil {
						panic(err)
					}
					err = dbx.Get(&arrivalStation, query, reservation.Arrival)
					if err != nil {
						panic(err)
					}

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

				seatInformationList = append(seatInformationList, s)
			}

			// 曖昧予約席とその他の候補席を選出
			var seatnum int           // 予約する座席の合計数
			var reserved bool         // あいまい指定席確保済フラグ
			var vargue bool           // あいまい検索フラグ
			var VagueSeat RequestSeat // あいまい指定席保存用
			reserved = false
			vargue = true
			seatnum = (req.Adult + req.Child - 1) // 全体の人数からあいまい指定席分を引いておく
			if req.Column == "" {                 // A/B/C/D/Eを指定しなければ、空いている適当な指定席を取るあいまいモード
				seatnum = (req.Adult + req.Child) // あいまい指定せず大人＋小人分の座席を取る
				reserved = true                   // dummy
				vargue = false                    // dummy
			}
			var CandidateSeat RequestSeat
			CandidateSeats := []RequestSeat{}

			// シート分だけ回して予約できる席を検索
			var i int
			for _, seat := range seatInformationList {
				if seat.Column == req.Column && !seat.IsOccupied && !reserved && vargue { // あいまい席があいてる
					VagueSeat.Row = seat.Row
					VagueSeat.Column = seat.Column
					reserved = true
				} else if !seat.IsOccupied && i < seatnum { // 単に席があいてる
					CandidateSeat.Row = seat.Row
					CandidateSeat.Column = seat.Column
					CandidateSeats = append(CandidateSeats, CandidateSeat)
					i++
				}
			}

			if vargue && reserved { // あいまい席が見つかり、予約できそうだった
				req.Seats = append(req.Seats, VagueSeat) // あいまい予約席を追加
			}
			if i > 0 { // 候補席があった
				req.Seats = append(req.Seats, CandidateSeats...) // 予約候補席追加
			}

			if len(req.Seats) < req.Adult+req.Child {
				// リクエストに対して席数が足りてない
				// 次の号車にうつしたい
				fmt.Println("-----------------")
				fmt.Printf("現在検索中の車両: %d号車, リクエスト座席数: %d, 予約できそうな座席数: %d, 不足数: %d\n", carnum, req.Adult+req.Child, len(req.Seats), req.Adult+req.Child-len(req.Seats))
				fmt.Println("リクエストに対して座席数が不足しているため、次の車両を検索します。")
				req.Seats = []RequestSeat{}
				if carnum == 16 {
					fmt.Println("この新幹線にまとめて予約できる席数がなかったから検索をやめるよ")
					req.Seats = []RequestSeat{}
					break
				}
			}
			fmt.Printf("空き実績: %d号車 シート:%v 席数:%d\n", carnum, req.Seats, len(req.Seats))
			if len(req.Seats) >= req.Adult+req.Child {
				fmt.Println("予約情報に追加したよ")
				req.Seats = req.Seats[:req.Adult+req.Child]
				req.CarNumber = carnum
				break
			}
		}
		if len(req.Seats) == 0 {
			errorResponse(w, http.StatusNotFound, "あいまい座席予約ができませんでした。指定した席、もしくは1車両内に希望の席数をご用意できませんでした。")
			return
		}
	default:
		// 座席情報のValidate
		seatList := Seat{}
		for _, z := range req.Seats {
			fmt.Println("XXXX", z)
			query = "SELECT * FROM seat_master WHERE train_class=? AND car_number=? AND seat_column=? AND seat_row=? AND seat_class=?"
			err = dbx.Get(
				&seatList, query,
				req.TrainClass,
				req.CarNumber,
				z.Column,
				z.Row,
				req.SeatClass,
			)
			if err != nil {
				errorResponse(w, http.StatusNotFound, "リクエストされた座席情報は存在しません。号車・喫煙席・座席クラスなど組み合わせを見直してください")
				log.Println(err.Error())
				return
			}
		}
		break
	}

	// 当該列車・列車名の予約一覧取得
	tx := dbx.MustBegin()
	reservations := []Reservation{}
	query = "SELECT * FROM reservations WHERE date=? AND train_class=? AND train_name=? FOR UPDATE"
	err = tx.Select(
		&reservations, query,
		date.Format("2006/01/02"),
		req.TrainClass,
		req.TrainName,
	)
	if err != nil {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, "列車予約情報の取得に失敗しました")
		log.Println(err.Error())
		return
	}

	for _, reservation := range reservations {
		if req.SeatClass == "non-reserved" {
			break
		}
		// train_masterから列車情報を取得(上り・下りが分かる)
		tmas = Train{}
		query = "SELECT * FROM train_master WHERE date=? AND train_class=? AND train_name=?"
		err = tx.Get(
			&tmas, query,
			date.Format("2006/01/02"),
			req.TrainClass,
			req.TrainName,
		)
		if err == sql.ErrNoRows {
			tx.Rollback()
			errorResponse(w, http.StatusNotFound, "列車データがみつかりません")
			log.Println(err.Error())
			return
		}
		if err != nil {
			tx.Rollback()
			errorResponse(w, http.StatusInternalServerError, "列車データの取得に失敗しました")
			log.Println(err.Error())
			return
		}

		// 予約情報の乗車区間の駅IDを求める
		var reservedfromStation, reservedtoStation Station
		query = "SELECT * FROM station_master WHERE name=?"

		// From
		err = tx.Get(&reservedfromStation, query, reservation.Departure)
		if err == sql.ErrNoRows {
			tx.Rollback()
			errorResponse(w, http.StatusNotFound, "予約情報に記載された列車の乗車駅データがみつかりません")
			log.Println(err.Error())
			return
		}
		if err != nil {
			tx.Rollback()
			errorResponse(w, http.StatusInternalServerError, "予約情報に記載された列車の乗車駅データの取得に失敗しました")
			log.Println(err.Error())
			return
		}

		// To
		err = tx.Get(&reservedtoStation, query, reservation.Arrival)
		if err == sql.ErrNoRows {
			tx.Rollback()
			errorResponse(w, http.StatusNotFound, "予約情報に記載された列車の降車駅データがみつかりません")
			log.Println(err.Error())
			return
		}
		if err != nil {
			tx.Rollback()
			errorResponse(w, http.StatusInternalServerError, "予約情報に記載された列車の降車駅データの取得に失敗しました")
			log.Println(err.Error())
			return
		}

		// 予約の区間重複判定
		secdup := false
		if tmas.IsNobori {
			// 上り
			if toStation.ID < reservedtoStation.ID && fromStation.ID <= reservedtoStation.ID {
				// pass
			} else if toStation.ID >= reservedfromStation.ID && fromStation.ID > reservedfromStation.ID {
				// pass
			} else {
				secdup = true
			}
		} else {
			// 下り
			if fromStation.ID < reservedfromStation.ID && toStation.ID <= reservedfromStation.ID {
				// pass
			} else if fromStation.ID >= reservedtoStation.ID && toStation.ID > reservedtoStation.ID {
				// pass
			} else {
				secdup = true
			}
		}

		if secdup {

			// 区間重複の場合は更に座席の重複をチェックする
			SeatReservations := []SeatReservation{}
			query := "SELECT * FROM seat_reservations WHERE reservation_id=? FOR UPDATE"
			err = tx.Select(
				&SeatReservations, query,
				reservation.ReservationId,
			)
			if err != nil {
				tx.Rollback()
				errorResponse(w, http.StatusInternalServerError, "座席予約情報の取得に失敗しました")
				log.Println(err.Error())
				return
			}

			for _, v := range SeatReservations {
				for _, seat := range req.Seats {
					if v.CarNumber == req.CarNumber && v.SeatRow == seat.Row && v.SeatColumn == seat.Column {
						tx.Rollback()
						fmt.Println("Duplicated ", reservation)
						errorResponse(w, http.StatusBadRequest, "リクエストに既に予約された席が含まれています")
						return
					}
				}
			}
		}
	}
	// 3段階の予約前チェック終わり

	// 自由席は強制的にSeats情報をダミーにする（自由席なのに席指定予約は不可）
	if req.SeatClass == "non-reserved" {
		req.Seats = []RequestSeat{}
		dummySeat := RequestSeat{}
		req.CarNumber = 0
		for num := 0; num < req.Adult+req.Child; num++ {
			dummySeat.Row = 0
			dummySeat.Column = ""
			req.Seats = append(req.Seats, dummySeat)
		}
	}

	// 運賃計算
	var fare int
	switch req.SeatClass {
	case "premium":
		fare, err = fareCalc(date, fromStation.ID, toStation.ID, req.TrainClass, "premium")
		if err != nil {
			tx.Rollback()
			errorResponse(w, http.StatusBadRequest, err.Error())
			log.Println("fareCalc " + err.Error())
			return
		}
	case "reserved":
		fare, err = fareCalc(date, fromStation.ID, toStation.ID, req.TrainClass, "reserved")
		if err != nil {
			tx.Rollback()
			errorResponse(w, http.StatusBadRequest, err.Error())
			log.Println("fareCalc " + err.Error())
			return
		}
	case "non-reserved":
		fare, err = fareCalc(date, fromStation.ID, toStation.ID, req.TrainClass, "non-reserved")
		if err != nil {
			tx.Rollback()
			errorResponse(w, http.StatusBadRequest, err.Error())
			log.Println("fareCalc " + err.Error())
			return
		}
	default:
		tx.Rollback()
		errorResponse(w, http.StatusBadRequest, "リクエストされた座席クラスが不明です")
		return
	}
	sumFare := (req.Adult * fare) + (req.Child*fare)/2
	fmt.Println("SUMFARE")

	// userID取得。ログインしてないと怒られる。
	user, errCode, errMsg := getUser(r)
	if errCode != http.StatusOK {
		tx.Rollback()
		errorResponse(w, errCode, errMsg)
		log.Printf("%s", errMsg)
		return
	}

	//予約ID発行と予約情報登録
	query = "INSERT INTO `reservations` (`user_id`, `date`, `train_class`, `train_name`, `departure`, `arrival`, `status`, `payment_id`, `adult`, `child`, `amount`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	result, err := tx.Exec(
		query,
		user.ID,
		date.Format("2006/01/02"),
		req.TrainClass,
		req.TrainName,
		req.Departure,
		req.Arrival,
		"requesting",
		"a",
		req.Adult,
		req.Child,
		sumFare,
	)
	if err != nil {
		tx.Rollback()
		errorResponse(w, http.StatusBadRequest, "予約の保存に失敗しました。"+err.Error())
		log.Println(err.Error())
		return
	}

	id, err := result.LastInsertId() //予約ID
	if err != nil {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, "予約IDの取得に失敗しました")
		log.Println(err.Error())
		return
	}

	//席の予約情報登録
	//reservationsレコード1に対してseat_reservationstが1以上登録される
	query = "INSERT INTO `seat_reservations` (`reservation_id`, `car_number`, `seat_row`, `seat_column`) VALUES (?, ?, ?, ?)"
	for _, v := range req.Seats {
		_, err = tx.Exec(
			query,
			id,
			req.CarNumber,
			v.Row,
			v.Column,
		)
		if err != nil {
			tx.Rollback()
			errorResponse(w, http.StatusInternalServerError, "座席予約の登録に失敗しました")
			log.Println(err.Error())
			return
		}
	}

	rr := TrainReservationResponse{
		ReservationId: id,
		Amount:        sumFare,
		IsOk:          true,
	}
	response, err := json.Marshal(rr)
	if err != nil {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, "レスポンスの生成に失敗しました")
		log.Println(err.Error())
		return
	}
	tx.Commit()
	w.Write(response)
}

func reservationPaymentHandler(w http.ResponseWriter, r *http.Request) {
	/*
		支払い及び予約確定API
		POST /api/train/reservation/commit
		{
			"card_token": "161b2f8f-791b-4798-42a5-ca95339b852b",
			"reservation_id": "1"
		}

		前段でフロントがクレカ非保持化対応用のpayment-APIを叩き、card_tokenを手に入れている必要がある
		レスポンスは成功か否かのみ返す
	*/

	// json parse
	req := new(ReservationPaymentRequest)
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "JSON parseに失敗しました")
		log.Println(err.Error())
		return
	}

	tx := dbx.MustBegin()

	// 予約IDで検索
	reservation := Reservation{}
	query := "SELECT * FROM reservations WHERE reservation_id=?"
	err = tx.Get(
		&reservation, query,
		req.ReservationId,
	)
	if err == sql.ErrNoRows {
		tx.Rollback()
		errorResponse(w, http.StatusNotFound, "予約情報がみつかりません")
		log.Println(err.Error())
		return
	}
	if err != nil {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, "予約情報の取得に失敗しました")
		log.Println(err.Error())
		return
	}

	// 支払い前のユーザチェック。本人以外のユーザの予約を支払ったりキャンセルできてはいけない。
	user, errCode, errMsg := getUser(r)
	if errCode != http.StatusOK {
		tx.Rollback()
		errorResponse(w, errCode, errMsg)
		log.Printf("%s", errMsg)
		return
	}
	if int64(*reservation.UserId) != user.ID {
		tx.Rollback()
		errorResponse(w, http.StatusForbidden, "他のユーザIDの支払いはできません")
		log.Println(err.Error())
		return
	}

	// 予約情報の支払いステータス確認
	switch reservation.Status {
	case "done":
		tx.Rollback()
		errorResponse(w, http.StatusForbidden, "既に支払いが完了している予約IDです")
		return
	default:
		break
	}

	// 決済する
	payInfo := PaymentInformationRequest{req.CardToken, req.ReservationId, reservation.Amount}
	j, err := json.Marshal(PaymentInformation{PayInfo: payInfo})
	if err != nil {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, "JSON Marshalに失敗しました")
		log.Println(err.Error())
		return
	}

	payment_api := os.Getenv("PAYMENT_API")
	if payment_api == "" {
		payment_api = "http://payment:5000"
	}

	resp, err := http.Post(payment_api+"/payment", "application/json", bytes.NewBuffer(j))
	if err != nil {
		tx.Rollback()
		errorResponse(w, resp.StatusCode, "HTTP POSTに失敗しました")
		log.Println(err.Error())
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, "レスポンスの読み込みに失敗しました")
		log.Println(err.Error())
		return
	}

	// リクエスト失敗
	if resp.StatusCode != http.StatusOK {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, "決済に失敗しました。カードトークンや支払いIDが間違っている可能性があります")
		log.Println(resp.StatusCode)
		return
	}

	// リクエスト取り出し
	output := PaymentResponse{}
	err = json.Unmarshal(body, &output)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "JSON parseに失敗しました")
		log.Println(err.Error())
		return
	}

	// 予約情報の更新
	query = "UPDATE reservations SET status=?, payment_id=? WHERE reservation_id=?"
	_, err = tx.Exec(
		query,
		"done",
		output.PaymentId,
		req.ReservationId,
	)
	if err != nil {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, "予約情報の更新に失敗しました")
		log.Println(err.Error())
		return
	}

	rr := ReservationPaymentResponse{
		IsOk: true,
	}
	response, err := json.Marshal(rr)
	if err != nil {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, "レスポンスの生成に失敗しました")
		log.Println(err.Error())
		return
	}
	tx.Commit()
	w.Write(response)
}

func signUpHandler(w http.ResponseWriter, r *http.Request) {
	/*
		ユーザー登録
		POST /auth/signup
	*/

	defer r.Body.Close()
	buf, _ := ioutil.ReadAll(r.Body)

	user := User{}
	json.Unmarshal(buf, &user)

	// TODO: validation

	salt := make([]byte, 1024)
	_, err := crand.Read(salt)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "salt generator error")
		return
	}
	superSecurePassword := pbkdf2.Key([]byte(user.Password), salt, 100, 256, sha256.New)

	_, err = dbx.Exec(
		"INSERT INTO `users` (`email`, `salt`, `super_secure_password`) VALUES (?, ?, ?)",
		user.Email,
		salt,
		superSecurePassword,
	)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "user registration failed")
		return
	}

	messageResponse(w, "registration complete")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	/*
		ログイン
		POST /auth/login
	*/

	defer r.Body.Close()
	buf, _ := ioutil.ReadAll(r.Body)

	postUser := User{}
	json.Unmarshal(buf, &postUser)

	user := User{}
	query := "SELECT * FROM users WHERE email=?"
	err := dbx.Get(&user, query, postUser.Email)
	if err == sql.ErrNoRows {
		errorResponse(w, http.StatusForbidden, "authentication failed")
		return
	}
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	challengePassword := pbkdf2.Key([]byte(postUser.Password), user.Salt, 100, 256, sha256.New)

	if !bytes.Equal(user.HashedPassword, challengePassword) {
		errorResponse(w, http.StatusForbidden, "authentication failed")
		return
	}

	session := getSession(r)

	session.Values["user_id"] = user.ID
	if err = session.Save(r, w); err != nil {
		log.Print(err)
		errorResponse(w, http.StatusInternalServerError, "session error")
		return
	}
	messageResponse(w, "autheticated")
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	/*
		ログアウト
		POST /auth/logout
	*/

	session := getSession(r)

	session.Values["user_id"] = 0
	if err := session.Save(r, w); err != nil {
		log.Print(err)
		errorResponse(w, http.StatusInternalServerError, "session error")
		return
	}
	messageResponse(w, "logged out")
}

func userReservationCancelHandler(w http.ResponseWriter, r *http.Request) {
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

	tx := dbx.MustBegin()

	reservation := Reservation{}
	query := "SELECT * FROM reservations WHERE reservation_id=? AND user_id=?"
	err = tx.Get(&reservation, query, itemID, user.ID)
	fmt.Println("CANCEL", reservation, itemID, user.ID)
	if err == sql.ErrNoRows {
		tx.Rollback()
		errorResponse(w, http.StatusBadRequest, "reservations naiyo")
		return
	}
	if err != nil {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, "予約情報の検索に失敗しました")
	}

	switch reservation.Status {
	case "rejected":
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, "何らかの理由により予約はRejected状態です")
		return
	case "done":
		// 支払いをキャンセルする
		payInfo := CancelPaymentInformationRequest{reservation.PaymentId}
		j, err := json.Marshal(payInfo)
		if err != nil {
			tx.Rollback()
			errorResponse(w, http.StatusInternalServerError, "JSON Marshalに失敗しました")
			log.Println(err.Error())
			return
		}

		payment_api := os.Getenv("PAYMENT_API")
		if payment_api == "" {
			payment_api = "http://payment:5000"
		}

		client := &http.Client{Timeout: time.Duration(10) * time.Second}
		req, err := http.NewRequest("DELETE", payment_api+"/payment/"+reservation.PaymentId, bytes.NewBuffer(j))
		if err != nil {
			tx.Rollback()
			errorResponse(w, http.StatusInternalServerError, "HTTPリクエストの作成に失敗しました")
			log.Println(err.Error())
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			tx.Rollback()
			errorResponse(w, resp.StatusCode, "HTTP DELETEに失敗しました")
			log.Println(err.Error())
			return
		}
		defer resp.Body.Close()

		// リクエスト失敗
		if resp.StatusCode != http.StatusOK {
			tx.Rollback()
			errorResponse(w, http.StatusInternalServerError, "決済のキャンセルに失敗しました")
			log.Println(resp.StatusCode)
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			tx.Rollback()
			errorResponse(w, http.StatusInternalServerError, "レスポンスの読み込みに失敗しました")
			log.Println(err.Error())
			return
		}

		// リクエスト取り出し
		output := CancelPaymentInformationResponse{}
		err = json.Unmarshal(body, &output)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "JSON parseに失敗しました")
			log.Println(err.Error())
			return
		}
		fmt.Println(output)
	default:
		// pass(requesting状態のものはpayment_id無いので叩かない)
	}

	query = "DELETE FROM reservations WHERE reservation_id=? AND user_id=?"
	_, err = tx.Exec(query, itemID, user.ID)
	if err != nil {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	query = "DELETE FROM seat_reservations WHERE reservation_id=?"
	_, err = tx.Exec(query, itemID)
	if err == sql.ErrNoRows {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, "seat naiyo")
		// errorResponse(w, http.Status, "authentication failed")
		return
	}
	if err != nil {
		tx.Rollback()
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	tx.Commit()
	messageResponse(w, "cancell complete")
}
