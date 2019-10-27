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
var initialTrains = func()[]Train{
	result := make([]Train,len(rawTrainData))
	startTime, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", "2020-01-01 00:00:00 +0800 CST")
	for i,raw := range rawTrainData{
		result[i] = Train{
			startTime.addDate(0,0,(i / 192)),
			raw.DepartureAt,
			raw.TrainClass,
			strconv.Itoa((i%192)+1),
			raw.StartStation,
			raw.LastStation,
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

