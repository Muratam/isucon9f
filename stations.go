package main

// ID は 1 少ないので注意
var initialStationsByID = []Station{
	Station{1, "東京", 0, true, true, true},
	Station{2, "古岡", 12.745608, false, true, true},
	Station{3, "絵寒町", 32.107649, false, false, true},
	Station{4, "沙芦公園", 45.037138, false, false, true},
	Station{5, "形顔", 52.773422, false, true, true},
	Station{6, "油交", 60.930427, true, true, true},
	Station{7, "通墨山", 72.915666, false, false, true},
	Station{8, "初野", 80.517696, false, true, true},
	Station{9, "樺威学園", 96.053004, false, true, true},
	Station{10, "塩鮫公園", 112.665386, false, true, true},
	Station{11, "山田", 119.444708, false, false, true},
	Station{12, "表岡", 131.462232, false, false, true},
	Station{13, "並取", 149.826976, false, false, true},
	Station{14, "細野", 166.909255, false, false, true},
	Station{15, "住郷", 182.323457, false, false, true},
	Station{16, "管英", 188.887999, false, false, true},
	Station{17, "気川", 207.599747, false, true, true},
	Station{18, "桐飛", 217.900353, false, false, true},
	Station{19, "樫曲町", 229.697609, false, false, true},
	Station{20, "依酒山", 244.77017, false, false, true},
	Station{21, "堀切町", 251.94859, false, false, true},
	Station{22, "葉千", 269.00928, false, false, true},
	Station{23, "奥山", 275.384825, false, false, true},
	Station{24, "鯉秋寺", 284.952294, false, false, true},
	Station{25, "伍出", 291.499545, false, false, true},
	Station{26, "杏高公園", 310.086023, false, false, true},
	Station{27, "荒川", 325.553902, true, true, true},
	Station{28, "磯川", 334.561908, false, false, true},
	Station{29, "茶川", 343.842013, false, true, true},
	Station{30, "八実学園", 355.192588, false, true, true},
	Station{31, "梓金", 374.584703, false, true, true},
	Station{32, "鯉田", 381.847874, false, true, true},
	Station{33, "鳴門", 393.244289, false, false, true},
	Station{34, "曲徳町", 411.802367, false, false, true},
	Station{35, "彩岬山", 420.375925, false, false, true},
	Station{36, "根永", 428.829478, false, true, true},
	Station{37, "鹿近川", 445.676144, false, false, true},
	Station{38, "結広", 457.246917, false, true, true},
	Station{39, "庵金公園", 474.044387, false, true, true},
	Station{40, "近岡", 487.270404, false, false, true},
	Station{41, "威香", 504.16358, false, false, true},
	Station{42, "名古屋", 519.612391, true, true, true},
	Station{43, "錦太学園", 531.408202, false, false, true},
	Station{44, "和錦台", 548.584849, false, false, true},
	Station{45, "稲冬台", 554.215596, false, false, true},
	Station{46, "松港山", 572.885503, false, false, true},
	Station{47, "甘桜", 584.344724, false, false, true},
	Station{48, "根左海岸", 603.713433, false, false, true},
	Station{49, "島威寺", 614.711098, false, false, true},
	Station{50, "月朱野", 633.406177, false, false, true},
	Station{51, "芋呉川", 640.097895, false, true, true},
	Station{52, "木南", 657.573946, false, false, true},
	Station{53, "鳩平ヶ丘", 677.211495, false, false, true},
	Station{54, "維荻学園", 689.581633, false, false, true},
	Station{55, "保池", 696.405431, false, true, true},
	Station{56, "九野", 711.087956, false, true, true},
	Station{57, "桜田", 728.268005, false, false, true},
	Station{58, "霞苑野", 735.983348, false, true, true},
	Station{59, "夷太寺", 744.58156, false, false, true},
	Station{60, "甘野", 751.340202, false, false, true},
	Station{61, "遠山", 770.125141, false, true, true},
	Station{62, "銀正", 788.163214, false, false, true},
	Station{63, "末国", 799.939778, false, false, true},
	Station{64, "泉別川", 807.476895, false, true, true},
	Station{65, "京都", 819.772794, true, true, true},
	Station{66, "桜内", 833.349255, false, true, true},
	Station{67, "荻葛ヶ丘", 839.29845, false, true, true},
	Station{68, "雨墨", 853.080719, false, true, true},
	Station{69, "桂綾寺", 863.842723, false, true, true},
	Station{70, "宇治", 869.266132, true, true, true},
	Station{71, "塚手海岸", 878.247393, false, true, true},
	Station{72, "垣通海岸", 893.724394, false, false, true},
	Station{73, "雨稲ヶ丘", 900.098745, false, true, true},
	Station{74, "森果川", 909.518544, true, true, true},
	Station{75, "舟田", 919.249073, false, false, true},
	Station{76, "形利", 938.540025, false, false, true},
	Station{77, "午万台", 954.151248, false, false, true},
	Station{78, "早森野", 966.498192, false, false, true},
	Station{79, "桐氷野", 975.568259, false, true, true},
	Station{80, "条川", 990.339004, true, true, true},
	Station{81, "菊岡", 1005.597665, false, true, true},
	Station{82, "大阪", 1024.983484, true, true, true},
}
var initialStationByIDDesc = func() []Station {
	result := make([]Station, len(initialStationsByID))
	for i := 0; i < len(initialStationsByID); i++ {
		result[i] = initialStationsByID[len(initialStationsByID)-i-1]
	}
	return result
}()
var getStationByName = func() map[string]Station {
	result := map[string]Station{}
	for _, station := range initialStationsByID {
		result[station.Name] = station
	}
	return result
}()

func getStationByID(id int) Station {
	return initialStationsByID[id-1]
}
