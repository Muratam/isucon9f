package main

import (
	"fmt"
	"time"
)

// arrival は固定.

// "SELECT  FROM train_timetable_master WHERE date=? AND train_class=? AND train_name=? AND station=?"
// 東京->大阪, 1: 東京->京都, 2:東京->名古屋
// date + train_name で絞れている([最速,中間,遅いやつ])
// のぞみ・こだま・ひかり の経過時間(+方向を)
// var x1 = []string{ // 最速:東京->大阪
// 	"06:01:00", "06:00:00",
// 	"06:09:18", "06:08:18",
// 	"06:42:03", "06:41:03",
// 	"07:06:21", "07:05:21",
// 	"07:43:22", "07:42:22",
// 	"07:50:18", "07:49:18",
// 	"07:56:08", "07:55:08",
// 	"08:06:50", "08:05:50",
// 	"08:11:59", "08:10:59",
// }
func getArrival(train Train, stationName string) time.Time {
	station := getStationByName[stationName]
	distance := station.Distance
	trainAverageSpeed := []float64{500, 480, 480}
	dt := distance / trainAverageSpeed[trainClassNameToIndex(train.TrainClass)] * 3600
	t, _ := time.Parse("15:04:05", train.DepartureAt)
	t2, _ := time.Parse("15:04:05", "00:00:00")
	start := train.Date.Add(t.Sub(t2))
	fmt.Println(dt)
	// 08:10:59
	return start.Add(time.Second * time.Duration(dt))
}

/*
+------------+-------------+------------+---------+-----------+----------+
| date       | train_class | train_name | station | departure | arrival  |
+------------+-------------+------------+---------+-----------+----------+
| 2020-01-01 | ??          | 1          | ??      | 06:01:00  | 06:00:00 |
| 2020-01-01 | ??          | 1          | ??      | 06:09:18  | 06:08:18 |
| 2020-01-01 | ??          | 1          | ??      | 06:42:03  | 06:41:03 |
| 2020-01-01 | ??          | 1          | ???     | 07:06:21  | 07:05:21 |
| 2020-01-01 | ??          | 1          | ??      | 07:43:22  | 07:42:22 |
| 2020-01-01 | ??          | 1          | ??      | 07:50:18  | 07:49:18 |
| 2020-01-01 | ??          | 1          | ???     | 07:56:08  | 07:55:08 |
| 2020-01-01 | ??          | 1          | ??      | 08:06:50  | 08:05:50 |
| 2020-01-01 | ??          | 1          | ??      | 08:11:59  | 08:10:59 |
+------------+-------------+------------+---------+-----------+----------+
*/

// Train{timeForinitialTrains("2020-01-01 00:00:00 +0800 CST"),"06:00:00","最速","1","東京","大阪",false,},
// Train{timeForinitialTrains("2020-01-01 00:00:00 +0800 CST"),"06:12:00","中間","3","東京","大阪",false,},
// Train{timeForinitialTrains("2020-01-01 00:00:00 +0800 CST"),"06:59:00","最速","13","東京","大阪",false,},
// Train{timeForinitialTrains("2020-01-01 00:00:00 +0800 CST"),"07:28:00","最速","19","東京","大阪",false,},
// Train{timeForinitialTrains("2020-01-01 00:00:00 +0800 CST"),"06:23:00","最速","6","大阪","東京",true,},
// Train{timeForinitialTrains("2020-01-01 00:00:00 +0800 CST"),"06:34:00","最速","8","大阪","東京",true,},
// Train{timeForinitialTrains("2020-01-01 00:00:00 +0800 CST"),"06:42:00","遅いやつ","10","大阪","東京",true,},
// Train{timeForinitialTrains("2020-01-01 00:00:00 +0800 CST"),"07:04:00","遅いやつ","14","大阪","東京",true,},
// Train{timeForinitialTrains("2020-01-01 00:00:00 +0800 CST"),"07:12:00","最速","16","大阪","東京",true,},
// Train{timeForinitialTrains("2020-01-01 00:00:00 +0800 CST"),"07:22:00","最速","18","大阪","東京",true,},
// Train{timeForinitialTrains("2020-01-01 00:00:00 +0800 CST"),"07:33:00","中間","20","大阪","東京",true,},
// Train{timeForinitialTrains("2020-01-01 00:00:00 +0800 CST"),"07:41:00","中間","22","大阪","東京",true,},
