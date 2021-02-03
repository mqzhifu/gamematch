package gamematch

func getErrorCode() (container []string) {
	//redis相关
	container = append(container, "300,redis connect err,error")
	container = append(container, "301,redisDo has err {0},error")
	//报名相关
	container = append(container, "400,ruleId not in redis db,error")
	container = append(container, "401,players len is : 0,error")
	container = append(container, "402,sign err Status:PlayerStatusSign  id:{0} ,error")
	container = append(container, "403,sign err Status:PlayerStatusSuccess  id:{0} ,error")
	container = append(container, "405,sign err Status:PlayerStatusSign id:{0} but timeout and grou person > 1 ,error")
	//rule
	container = append(container, "600,rule confg HGETALL err:{0},error")
	container = append(container, "601,RuleConfig is null  ,error")
	container = append(container, "602,rule id not exist :{0},error")
	container = append(container, "603,addRuleOne redis err,error")

	container = append(container, "604,id <= 0,error")
	container = append(container, "605,AppId <= 0,error")
	container = append(container, "606,Flag <= 0,error")
	container = append(container, "607,Flag must : 1 or 2,error")
	container = append(container, "608,TeamVSPerson <= 0,error")
	container = append(container, "609,TeamVSPerson > {0},error")
	container = append(container, "610,PersonCondition <= 0,error")
	container = append(container, "611,PersonCondition > {0},error")
	container = append(container, "612,MatchTimeout < {0} or MatchTimeout > {1},error")
	container = append(container, "613,SuccessTimeout < {0} or SuccessTimeout > {1},error")
	container = append(container, "614,GroupPersonMax <= 0,error")
	container = append(container, "615,GroupPersonMax > {0},error")
	container = append(container, "616,CategoryKey is empty,error")

	return container
}