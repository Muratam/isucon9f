package main

import "time"

func timeForinitialUsers(s string) time.Time {
	result, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", s)
	return result
}
func timeForinitialUsersPtr(s string) *time.Time {
	result, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", s)
	return &result
}

var initialDistanceFares = []DistanceFare{
	DistanceFare{0, 2500},
	DistanceFare{50, 3000},
	DistanceFare{75, 3700},
	DistanceFare{100, 4500},
	DistanceFare{150, 5200},
	DistanceFare{200, 6000},
	DistanceFare{300, 7200},
	DistanceFare{400, 8300},
	DistanceFare{500, 12000},
	DistanceFare{1000, 20000},
}
