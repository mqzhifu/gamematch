package gamematch

import (
	"src/zlib"
)
var AddRuleFlag = 0
func Test(){
	//AddRuleFlag = 1

	myGamematch := NewSelf()

	//total(100)

	//clear(*myGamematch)


	//myGamematch.DemonAllRuleCheckSignTimeout()

	//rule,_ := myGamematch.RuleConfig.GetById(2)
	//myGamematch.CheckSignTimeout(rule)

	//TestAddRuleData(*myGamematch)
	//TestSign(*myGamematch)
	//TestMatching(*myGamematch)

	matchingCase1(*myGamematch)

	zlib.ExitPrint(-999)
}

//func total(n int){
//	inc := 0
//	for a:=0;a<=n;a++{
//		for b:=0;b<=n/2;b++{
//			for c:=0;c<=n/5;c++{
//				if a + b + c == 100 {
//					inc++
//					zlib.MyPrint(a,"+",b,"+",c,"=100")
//				}
//			}
//		}
//	}
//	zlib.ExitPrint(inc)
//}




func clear(myGamematch Gamematch){
	myGamematch.DelAll()
	zlib.ExitPrint("clear all done. ")
}
func TestAddRuleData(myGamematch Gamematch){
	rule1 := Rule{
		Id: 1,
		AppId: 3,
		CategoryKey: "RuleFlagCollectPerson",
		MatchTimeout: 7,
		SuccessTimeout: 60,
		IsSupportGroup: 1,
		Flag: 2,
		PersonCondition: 4,
		GroupPersonMax:5,
		//TeamPerson: 5,
		PlayerWeight: PlayerWeight{
			ScoreMin:-1,
			ScoreMax:-1,
			AutoAssign:true,
			Formula:"",
			Flag:1,
		},
	}

	rule2 := Rule{
		Id: 2,
		AppId: 4,
		CategoryKey: "RuleFlagTeamVS",
		MatchTimeout: 10,
		SuccessTimeout: 70,
		IsSupportGroup: 1,
		Flag: 1,
		PersonCondition: 5,
		TeamVSPerson:5,
		GroupPersonMax:5,
		//TeamPerson: 5,
		PlayerWeight: PlayerWeight{
			ScoreMin:-1,
			ScoreMax:-1,
			AutoAssign:true,
			Formula:"",
			Flag:1,
		},
	}

	myGamematch.RuleConfig.AddOne(rule1)
	myGamematch.RuleConfig.AddOne(rule2)
}

func TestMatching(myGamematch Gamematch){
	//myGamematch.DemonStartAllRuleMatching()
}

func matchingCase1(myGamematch Gamematch){
	//myGamematch.testSignDel(2)
	//TestSign(myGamematch)
	myGamematch.DemonAll()
}

func TestSign(myGamematch Gamematch){
	ruleId := 2
	playerIdInc := 1
	signRuleArr := []int{1,5,4,5,3,3,3,2,2,1,1,1,1,5,4,4}
	for _,playerNumMax := range signRuleArr{
		var playerStructArr []Player
		for i:=0;i<playerNumMax;i++{
			player := Player{Id:playerIdInc}
			playerStructArr = append(playerStructArr,player)
			playerIdInc++
		}
		myGamematch.Sign(ruleId,playerStructArr)
	}



	//players =[]Player{
	//	Player{Id:2},
	//	Player{Id:3},
	//}
	//myGamematch.Sign(1,players)
	//
	//players =[]Player{
	//	Player{Id:4},
	//	Player{Id:5},
	//	Player{Id:6},
	//}
	//myGamematch.Sign(1,players)
	//
	//players =[]Player{
	//	Player{Id:7},
	//	Player{Id:8},
	//	Player{Id:9},
	//}
	//myGamematch.Sign(1,players)
	//
	//
	//players =[]Player{
	//	Player{Id:10},
	//}
	//myGamematch.Sign(1,players)
	//
	//players =[]Player{
	//	Player{Id:11},
	//}
	//myGamematch.Sign(1,players)
	//
	//players =[]Player{
	//	Player{Id:12},
	//	Player{Id:13},
	//}
	//myGamematch.Sign(1,players)
	//
	//players =[]Player{
	//	Player{Id:14},
	//	Player{Id:15},
	//}
	//myGamematch.Sign(1,players)
	//
	//
	//players =[]Player{
	//	Player{Id:16},
	//	Player{Id:17},
	//	Player{Id:18},
	//}
	//myGamematch.Sign(1,players)
	//
	//
	//players =[]Player{
	//	Player{Id:19},
	//}
	//myGamematch.Sign(1,players)
	//
	//players =[]Player{
	//	Player{Id:20},
	//	Player{Id:21},
	//}
	//myGamematch.Sign(1,players)
	//
	//
	//
	//players =[]Player{
	//	Player{Id:20},
	//	Player{Id:21},
	//	Player{Id:20},
	//	Player{Id:21},
	//	Player{Id:20},
	//}
	//myGamematch.Sign(1,players)
	//
	//
	//players =[]Player{
	//	Player{Id:20},
	//	Player{Id:21},
	//	Player{Id:20},
	//	Player{Id:21},
	//	Player{Id:20},
	//}
	//myGamematch.Sign(1,players)
	//
	//players =[]Player{
	//	Player{Id:20},
	//	Player{Id:21},
	//	Player{Id:20},
	//	Player{Id:21},
	//	Player{Id:20},
	//}
	//myGamematch.Sign(1,players)
	//
	//
	//


}

