package main
import (
	"time"
	"fmt"
	"strconv"
)
type Train struct {
	Date         time.Time `json:"date" db:"date"`
	DepartureAt  string    `json:"departure_at" db:"departure_at"`
	TrainClass   string    `json:"train_class" db:"train_class"`
	TrainName    string    `json:"train_name" db:"train_name"`
	StartStation string    `json:"start_station" db:"start_station"`
	LastStation  string    `json:"last_station" db:"last_station"`
	IsNobori     bool      `json:"is_nobori" db:"is_nobori"`
}

// 0: "東京->大阪", 1: "東京->京都", 2:"東京->名古屋" (start->last)
// +3 すると逆

var initialTrains = func()[]Train{
	result := make([]Train,len(RTNData))
	now, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", "2020-01-01 00:00:00 +0800 CST")
	fromTrainClassI := []string{"最速","中間","遅いやつ"}
	startStationFromStationsI := []string{"東京","東京","東京","大阪","京都","名古屋"}
	lastStationFromStationsI := []string{"大阪","京都","名古屋","東京","東京","東京"}
	for i,raw := range RTNData{
		if i % 192 == 191{
			now = now.AddDate(0,0,1)
		}
		result[i] = Train{
			now,
			raw.DepartureAt,
			fromTrainClassI[raw.TrainClassI],
			strconv.Itoa((i%192)+1),
			startStationFromStationsI[raw.StationsI],
			lastStationFromStationsI[raw.StationsI],
			i % 2 == 1,
		}
	}
	return result
}()

func main(){
	fmt.Println(len(initialTrains))
	for i, train := range initialTrains{
		if (i % 2 == 0 && train.IsNobori) || ((i % 2 != 0 && !train.IsNobori)) {
			fmt.Println("invalid")
		}
	}
}

