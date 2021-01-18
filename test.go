package gamematch

import "src/zlib"

func Test(){
	myGamematch := NewSelf()

	//testStruct := PlayerStatusElement{
	//	PlayerId :	1,
	//	Status 	:	2,
	//	RuleId	:	3,
	//	Weight	:	"aaa",
	//	GroupId	:	4,
	//	ATime 	:	5555,
	//	UTime 	:	6666,
	//}

	//rs := zlib.StructCovertMap(testStruct)
	//mymap := rs.(map[string]interface{})
	//playerStatusElement := &PlayerStatusElement{}
	//zlib.MyPrint(rs)
	//zlib.MapCovertStruct(mymap,playerStatusElement)
	//fmt.Println(fmt.Sprintf("%+v", playerStatusElement))
	//os.Exit(-100)

	//clear(*myGamematch)

	//TestAddRuleData(*myGamematch)
	//TestSign(*myGamematch)
	TestMatching(*myGamematch)

	zlib.ExitPrint(-500)
}
func clear(myGamematch Gamematch){
	myGamematch.DelAll()
	zlib.ExitPrint("clear all done. ")
}
func TestAddRuleData(myGamematch Gamematch){
	rule := Rule{
		Id: 1,
		AppId: 3,
		CategoryKey: "normal",
		MatchTimeout: 7,
		SuccessTimeout: 8,
		IsSupportGroup: 1,
		Flag: 2,
		PersonCondition: 4,
		TeamPerson: 5,
		PlayerWeight: PlayerWeight{
			ScoreMin:-1,
			ScoreMax:-1,
			AutoAssign:true,
			Formula:"",
			Flag:1,
		},
	}
	myGamematch.RuleConfig.AddOne(rule)
}

func TestMatching(myGamematch Gamematch){
	myGamematch.StartAllQueueMatching()
}

func TestSign(myGamematch Gamematch){
	players :=[]*Player{
		&Player{Id:1},
	}
	myGamematch.Sign(1,players)


	players =[]*Player{
		&Player{Id:2},
		&Player{Id:3},
	}
	myGamematch.Sign(1,players)

	players =[]*Player{
		&Player{Id:4},
		&Player{Id:5},
		&Player{Id:6},
	}
	myGamematch.Sign(1,players)

	players =[]*Player{
		&Player{Id:7},
		&Player{Id:8},
		&Player{Id:9},
	}
	myGamematch.Sign(1,players)


	players =[]*Player{
		&Player{Id:10},
	}
	myGamematch.Sign(1,players)

	players =[]*Player{
		&Player{Id:11},
	}
	myGamematch.Sign(1,players)

	players =[]*Player{
		&Player{Id:12},
		&Player{Id:13},
	}
	myGamematch.Sign(1,players)

	players =[]*Player{
		&Player{Id:14},
		&Player{Id:15},
	}
	myGamematch.Sign(1,players)


	players =[]*Player{
		&Player{Id:16},
		&Player{Id:17},
		&Player{Id:18},
	}
	myGamematch.Sign(1,players)


	players =[]*Player{
		&Player{Id:19},
	}
	myGamematch.Sign(1,players)

	players =[]*Player{
		&Player{Id:20},
		&Player{Id:21},
	}
	myGamematch.Sign(1,players)







}

