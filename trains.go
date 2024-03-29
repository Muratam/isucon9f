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

// date=? AND train_class IN (?) AND is_nobori=?
func searchTrain(date time.Time, trainClass []string, is_nobori bool) []Train {
	if date.Year() != 2020 {
		return []Train{}
	}
	result := []Train{}
	day := int(date.Sub(startDay2020).Hours()) / 24
	for i := 0; i < 192; i++ {
		if (is_nobori && i%2 == 0) || (!is_nobori && i%2 == 1) {
			continue
		}
		train := getTrainRaw(day, i)
		ok := false
		for _, cl := range trainClass {
			if train.TrainClass == cl {
				ok = true
			}
		}
		if !ok {
			continue
		}
		result = append(result, train)
	}
	return result
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
