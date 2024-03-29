package main

import "time"

func timeForinitialUsers(s string) time.Time {
	result, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", s)
	return result
}

var initialFares = []Fare{
	Fare{"最速", "premium", timeForinitialUsers("2020-01-01 00:00:00 +0800 CST"), 15},
	Fare{"最速", "reserved", timeForinitialUsers("2020-01-01 00:00:00 +0800 CST"), 9.375},
	Fare{"最速", "non-reserved", timeForinitialUsers("2020-01-01 00:00:00 +0800 CST"), 7.5},
	Fare{"中間", "premium", timeForinitialUsers("2020-01-01 00:00:00 +0800 CST"), 10},
	Fare{"中間", "reserved", timeForinitialUsers("2020-01-01 00:00:00 +0800 CST"), 6.25},
	Fare{"中間", "non-reserved", timeForinitialUsers("2020-01-01 00:00:00 +0800 CST"), 5},
	Fare{"遅いやつ", "premium", timeForinitialUsers("2020-01-01 00:00:00 +0800 CST"), 8},
	Fare{"遅いやつ", "reserved", timeForinitialUsers("2020-01-01 00:00:00 +0800 CST"), 5},
	Fare{"遅いやつ", "non-reserved", timeForinitialUsers("2020-01-01 00:00:00 +0800 CST"), 4},
	Fare{"最速", "premium", timeForinitialUsers("2020-01-06 00:00:00 +0800 CST"), 3},
	Fare{"最速", "reserved", timeForinitialUsers("2020-01-06 00:00:00 +0800 CST"), 1.875},
	Fare{"最速", "non-reserved", timeForinitialUsers("2020-01-06 00:00:00 +0800 CST"), 1.5},
	Fare{"中間", "premium", timeForinitialUsers("2020-01-06 00:00:00 +0800 CST"), 2},
	Fare{"中間", "reserved", timeForinitialUsers("2020-01-06 00:00:00 +0800 CST"), 1.25},
	Fare{"中間", "non-reserved", timeForinitialUsers("2020-01-06 00:00:00 +0800 CST"), 1},
	Fare{"遅いやつ", "premium", timeForinitialUsers("2020-01-06 00:00:00 +0800 CST"), 1.6},
	Fare{"遅いやつ", "reserved", timeForinitialUsers("2020-01-06 00:00:00 +0800 CST"), 1},
	Fare{"遅いやつ", "non-reserved", timeForinitialUsers("2020-01-06 00:00:00 +0800 CST"), 0.8},
	Fare{"最速", "premium", timeForinitialUsers("2020-03-13 00:00:00 +0800 CST"), 9},
	Fare{"最速", "reserved", timeForinitialUsers("2020-03-13 00:00:00 +0800 CST"), 5.625},
	Fare{"最速", "non-reserved", timeForinitialUsers("2020-03-13 00:00:00 +0800 CST"), 4.5},
	Fare{"中間", "premium", timeForinitialUsers("2020-03-13 00:00:00 +0800 CST"), 6},
	Fare{"中間", "reserved", timeForinitialUsers("2020-03-13 00:00:00 +0800 CST"), 3.75},
	Fare{"中間", "non-reserved", timeForinitialUsers("2020-03-13 00:00:00 +0800 CST"), 3},
	Fare{"遅いやつ", "premium", timeForinitialUsers("2020-03-13 00:00:00 +0800 CST"), 4.8},
	Fare{"遅いやつ", "reserved", timeForinitialUsers("2020-03-13 00:00:00 +0800 CST"), 3},
	Fare{"遅いやつ", "non-reserved", timeForinitialUsers("2020-03-13 00:00:00 +0800 CST"), 2.4},
	Fare{"最速", "premium", timeForinitialUsers("2020-04-01 00:00:00 +0800 CST"), 3},
	Fare{"最速", "reserved", timeForinitialUsers("2020-04-01 00:00:00 +0800 CST"), 1.875},
	Fare{"最速", "non-reserved", timeForinitialUsers("2020-04-01 00:00:00 +0800 CST"), 1.5},
	Fare{"中間", "premium", timeForinitialUsers("2020-04-01 00:00:00 +0800 CST"), 2},
	Fare{"中間", "reserved", timeForinitialUsers("2020-04-01 00:00:00 +0800 CST"), 1.25},
	Fare{"中間", "non-reserved", timeForinitialUsers("2020-04-01 00:00:00 +0800 CST"), 1},
	Fare{"遅いやつ", "premium", timeForinitialUsers("2020-04-01 00:00:00 +0800 CST"), 1.6},
	Fare{"遅いやつ", "reserved", timeForinitialUsers("2020-04-01 00:00:00 +0800 CST"), 1},
	Fare{"遅いやつ", "non-reserved", timeForinitialUsers("2020-04-01 00:00:00 +0800 CST"), 0.8},
	Fare{"最速", "premium", timeForinitialUsers("2020-04-24 00:00:00 +0800 CST"), 15},
	Fare{"最速", "reserved", timeForinitialUsers("2020-04-24 00:00:00 +0800 CST"), 9.375},
	Fare{"最速", "non-reserved", timeForinitialUsers("2020-04-24 00:00:00 +0800 CST"), 7.5},
	Fare{"中間", "premium", timeForinitialUsers("2020-04-24 00:00:00 +0800 CST"), 10},
	Fare{"中間", "reserved", timeForinitialUsers("2020-04-24 00:00:00 +0800 CST"), 6.25},
	Fare{"中間", "non-reserved", timeForinitialUsers("2020-04-24 00:00:00 +0800 CST"), 5},
	Fare{"遅いやつ", "premium", timeForinitialUsers("2020-04-24 00:00:00 +0800 CST"), 8},
	Fare{"遅いやつ", "reserved", timeForinitialUsers("2020-04-24 00:00:00 +0800 CST"), 5},
	Fare{"遅いやつ", "non-reserved", timeForinitialUsers("2020-04-24 00:00:00 +0800 CST"), 4},
	Fare{"最速", "premium", timeForinitialUsers("2020-05-11 00:00:00 +0800 CST"), 3},
	Fare{"最速", "reserved", timeForinitialUsers("2020-05-11 00:00:00 +0800 CST"), 1.875},
	Fare{"最速", "non-reserved", timeForinitialUsers("2020-05-11 00:00:00 +0800 CST"), 1.5},
	Fare{"中間", "premium", timeForinitialUsers("2020-05-11 00:00:00 +0800 CST"), 2},
	Fare{"中間", "reserved", timeForinitialUsers("2020-05-11 00:00:00 +0800 CST"), 1.25},
	Fare{"中間", "non-reserved", timeForinitialUsers("2020-05-11 00:00:00 +0800 CST"), 1},
	Fare{"遅いやつ", "premium", timeForinitialUsers("2020-05-11 00:00:00 +0800 CST"), 1.6},
	Fare{"遅いやつ", "reserved", timeForinitialUsers("2020-05-11 00:00:00 +0800 CST"), 1},
	Fare{"遅いやつ", "non-reserved", timeForinitialUsers("2020-05-11 00:00:00 +0800 CST"), 0.8},
	Fare{"最速", "premium", timeForinitialUsers("2020-08-07 00:00:00 +0800 CST"), 9},
	Fare{"最速", "reserved", timeForinitialUsers("2020-08-07 00:00:00 +0800 CST"), 5.625},
	Fare{"最速", "non-reserved", timeForinitialUsers("2020-08-07 00:00:00 +0800 CST"), 4.5},
	Fare{"中間", "premium", timeForinitialUsers("2020-08-07 00:00:00 +0800 CST"), 6},
	Fare{"中間", "reserved", timeForinitialUsers("2020-08-07 00:00:00 +0800 CST"), 3.75},
	Fare{"中間", "non-reserved", timeForinitialUsers("2020-08-07 00:00:00 +0800 CST"), 3},
	Fare{"遅いやつ", "premium", timeForinitialUsers("2020-08-07 00:00:00 +0800 CST"), 4.8},
	Fare{"遅いやつ", "reserved", timeForinitialUsers("2020-08-07 00:00:00 +0800 CST"), 3},
	Fare{"遅いやつ", "non-reserved", timeForinitialUsers("2020-08-07 00:00:00 +0800 CST"), 2.4},
	Fare{"最速", "premium", timeForinitialUsers("2020-08-24 00:00:00 +0800 CST"), 3},
	Fare{"最速", "reserved", timeForinitialUsers("2020-08-24 00:00:00 +0800 CST"), 1.875},
	Fare{"最速", "non-reserved", timeForinitialUsers("2020-08-24 00:00:00 +0800 CST"), 1.5},
	Fare{"中間", "premium", timeForinitialUsers("2020-08-24 00:00:00 +0800 CST"), 2},
	Fare{"中間", "reserved", timeForinitialUsers("2020-08-24 00:00:00 +0800 CST"), 1.25},
	Fare{"中間", "non-reserved", timeForinitialUsers("2020-08-24 00:00:00 +0800 CST"), 1},
	Fare{"遅いやつ", "premium", timeForinitialUsers("2020-08-24 00:00:00 +0800 CST"), 1.6},
	Fare{"遅いやつ", "reserved", timeForinitialUsers("2020-08-24 00:00:00 +0800 CST"), 1},
	Fare{"遅いやつ", "non-reserved", timeForinitialUsers("2020-08-24 00:00:00 +0800 CST"), 0.8},
	Fare{"最速", "premium", timeForinitialUsers("2020-12-25 00:00:00 +0800 CST"), 15},
	Fare{"最速", "reserved", timeForinitialUsers("2020-12-25 00:00:00 +0800 CST"), 9.375},
	Fare{"最速", "non-reserved", timeForinitialUsers("2020-12-25 00:00:00 +0800 CST"), 7.5},
	Fare{"中間", "premium", timeForinitialUsers("2020-12-25 00:00:00 +0800 CST"), 10},
	Fare{"中間", "reserved", timeForinitialUsers("2020-12-25 00:00:00 +0800 CST"), 6.25},
	Fare{"中間", "non-reserved", timeForinitialUsers("2020-12-25 00:00:00 +0800 CST"), 5},
	Fare{"遅いやつ", "premium", timeForinitialUsers("2020-12-25 00:00:00 +0800 CST"), 8},
	Fare{"遅いやつ", "reserved", timeForinitialUsers("2020-12-25 00:00:00 +0800 CST"), 5},
	Fare{"遅いやつ", "non-reserved", timeForinitialUsers("2020-12-25 00:00:00 +0800 CST"), 4},
}

// type Fare struct {
// 	TrainClass     string    `json:"train_class" db:"train_class"`
// 	SeatClass      string    `json:"seat_class" db:"seat_class"`
// 	StartDate      time.Time `json:"start_date" db:"start_date"`
// 	FareMultiplier float64   `json:"fare_multiplier" db:"fare_multiplier"`
// }
var FaresfromtrainClassSeatClass = func() map[string][]Fare {
	result := map[string][]Fare{}
	for _, fare := range initialFares {
		key := fare.TrainClass + fare.SeatClass
		val, ok := result[key]
		if !ok {
			val = []Fare{}
		}
		val = append(val, fare)
		result[key] = val
	}
	return result
}()
