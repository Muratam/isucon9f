package main

import (
	"errors"
	"strconv"
	"time"
)

// 0: "東京->大阪", 1: "東京->京都", 2:"東京->名古屋" (start->last)
// +3 すると逆
//{2020-01-01 00:00:00 +0800 CST 06:00:00 最速 1 東京 大阪 false}
//{2020-01-01 00:00:00 +0800 CST 06:00:00 遅いやつ 2 京都 東京 true}
//{2020-01-01 00:00:00 +0800 CST 06:12:00 中間 3 東京 大阪 false}
//{2020-01-01 00:00:00 +0800 CST 06:11:00 遅いやつ 4 名古屋 東京 true}
var startDay2020 = func() time.Time {
	result, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", "2020-01-01 00:00:00 +0800 CST")
	return result
}()
var fromTrainClassI = []string{"最速", "中間", "遅いやつ"}
var startStationFromStationsI = []string{"東京", "東京", "東京", "大阪", "京都", "名古屋"}
var lastStationFromStationsI = []string{"大阪", "京都", "名古屋", "東京", "東京", "東京"}

func getTrainRaw(dayI int, name int) Train {
	raw := RTNData[dayI][name]
	return Train{
		startDay2020.AddDate(0, 0, dayI),
		raw.DepartureAt + ":00",
		fromTrainClassI[raw.TrainClassI],
		strconv.Itoa(name + 1),
		startStationFromStationsI[raw.StationsI],
		lastStationFromStationsI[raw.StationsI],
		name%2 == 1,
	}
}

// date=? AND train_class=? AND train_name=?
//
func getTrain(date time.Time, trainnameStr string) (Train, error) {
	if date.Year() != 2020 {
		return Train{}, errors.New("invalid date:" + date.String() + trainnameStr)
	}
	trainname, err := strconv.Atoi(trainnameStr)
	if err != nil {
		return Train{}, err
	}
	if trainname <= 0 || trainname > 192 {
		return Train{}, errors.New("invalid train")
	}
	duration := int(date.Sub(startDay2020).Hours()) / 24
	return getTrainRaw(duration, trainname-1), nil
}

// name -> class は固定ではないので
func getTrainWithClass(date time.Time, trainnameStr string, trainclassStr string) (Train, error) {
	train, err := getTrain(date, trainnameStr)
	if err != nil {
		return train, err
	}
	if train.TrainClass != trainclassStr {
		return train, errors.New("invalid train class:" + train.TrainClass + " is not " + trainclassStr)
	}
	return train, nil
}

// var initialTrains = func() []Train {
// 	result := make([]Train, 70272)
// 	i := 0
// 	for dayI, raws := range RTNData {
// 		for name, _ := range raws {
// 			result[i] = getTrainRaw(dayI, name)
// 			i += 1
// 		}
// 	}
// 	return result
// }()
