package gamematch

//func (gamematch *Gamematch)CheckHttpRuleAddOneData(jsonDataMap map[string]string)(data SignCancelHttpData,errs error){
//matchCode,ok := jsonDataMap["matchCode"]
//if !ok {
//	return data,myerr.NewErrorCode(450)
//}
//if matchCode == ""{
//	return data,myerr.NewErrorCode(450)
//}
//
//checkCodeRs := false
//ruleId := 0
//for _,rule := range gamematch.RuleConfig.getAll(){
//	if  rule.CategoryKey == matchCode{
//		ruleId = rule.Id
//		checkCodeRs = true
//		break
//	}
//}
//if !checkCodeRs{
//	return data,myerr.NewErrorCode(451)
//}
//
//playerId,ok := jsonDataMap["playerId"]
//if !ok {
//	return data,myerr.NewErrorCode(452)
//}
//if playerId == ""{
//	return data,myerr.NewErrorCode(452)
//}
//
//data.PlayerId =  zlib.Atoi(playerId)
//data.RuleId = ruleId
//	return data,nil
//}

func (gamematch *Gamematch)CheckHttpSignCancelData(httpReqSignCancel HttpReqSignCancel)(data HttpReqSignCancel,errs error){
	//matchCode,ok := jsonDataMap["matchCode"]
	//if !ok {
	//	return data,myerr.NewErrorCode(450)
	//}
	if httpReqSignCancel.MatchCode == ""{
		return data,myerr.NewErrorCode(450)
	}

	rule ,err := gamematch.RuleConfig.getByCategory(httpReqSignCancel.MatchCode)
	if err !=nil{
		return data,err
	}
	if httpReqSignCancel.GroupId == 0 && httpReqSignCancel.PlayerId == 0 {
		return data,myerr.NewErrorCode(457)
	}

	httpReqSignCancel.RuleId = rule.Id
	data = httpReqSignCancel
	return data,nil

}
func (gamematch *Gamematch)CheckHttpSignData(httpReqSign HttpReqSign)(returnHttpReqSign HttpReqSign,errs error){
	if httpReqSign.MatchCode == ""{
		return returnHttpReqSign,myerr.NewErrorCode(450)
	}
	//checkCodeRs := false
	//ruleId := 0
	//for _,rule := range gamematch.RuleConfig.getAll(){
	//	if  rule.CategoryKey == httpReqSign.MatchCode{
	//		ruleId = rule.Id
	//		checkCodeRs = true
	//		break
	//	}
	//}
	//if !checkCodeRs{
	//	return returnHttpReqSign,myerr.NewErrorCode(451)
	//}
	rule ,err := gamematch.RuleConfig.getByCategory(httpReqSign.MatchCode)
	if err !=nil{
		return returnHttpReqSign,err
	}


	////groupId := zlib.Atoi(groupIdStr.(string))
	//groupId := int(groupIdStr.(float64))
	if httpReqSign.GroupId == 0{
		return returnHttpReqSign,myerr.NewErrorCode(452)
	}
	var playerListStruct []Player
	for _,v := range httpReqSign.PlayerList{
		if v.Uid == 0{
			return returnHttpReqSign,myerr.NewErrorCode(456)
		}
		playerListStruct = append(playerListStruct,Player{Id:v.Uid})
	}
	httpReqSign.RuleId = rule.Id
	returnHttpReqSign = httpReqSign

	return returnHttpReqSign,nil
}

//func (gamematch *Gamematch)CheckHttpSignDataOld(jsonDataMap map[string]interface{})(data SignHttpData,errs error){
//	httpReqSign := &HttpReqSign{}
//	zlib.MapCovertStruct(jsonDataMap,httpReqSign)
//	zlib.ExitPrint(httpReqSign)
//
//	mylog.Debug("CheckHttpSignData",jsonDataMap)
//	matchCodeStr,ok := jsonDataMap["matchCode"]
//	if !ok {
//		return data,myerr.NewErrorCode(450)
//	}
//	matchCode := matchCodeStr.(string)
//	if matchCode == ""{
//		return data,myerr.NewErrorCode(450)
//	}
//
//
//	checkCodeRs := false
//	ruleId := 0
//	for _,rule := range gamematch.RuleConfig.getAll(){
//		if  rule.CategoryKey == matchCode{
//			ruleId = rule.Id
//			checkCodeRs = true
//			break
//		}
//	}
//	if !checkCodeRs{
//		return data,myerr.NewErrorCode(451)
//	}
//
//	groupIdStr,ok := jsonDataMap["groupId"]
//	if !ok {
//		return data,myerr.NewErrorCode(452)
//	}
//	//fmt.Printf("type:%T, value:%+v\n", groupIdStr,groupIdStr)
//	//groupId := zlib.Atoi(groupIdStr.(string))
//	groupId := int(groupIdStr.(float64))
//	if groupId == 0{
//		return data,myerr.NewErrorCode(452)
//	}
//
//	customProp := ""
//	customPropStr,ok := jsonDataMap["customProp"]
//	if !ok {
//		//return data,err.NewErrorCode(453)
//		customProp = ""
//	}else{
//		customProp = customPropStr.(string)
//	}
//
//	playerListMap,ok := jsonDataMap["playerList"]
//	if !ok {
//		return data,myerr.NewErrorCode(454)
//	}
//
////[     map[matchAttr:map[age:10 sex:girl] uid:4] map[matchAttr:map[age:15 sex:man] uid:5]]
//	//fmt.Printf("unexpected type %T", playerListMap)
//	//fmt.Printf("playerListMap : %+v",playerListMap)
//	playerList ,ok := playerListMap.([]interface{})
//	if !ok{
//		return data,myerr.NewErrorCode(455)
//	}
//
//	var playerListStruct []Player
//	for _,v := range playerList{
//		row := v.(map[string]interface{})
//		zlib.MyPrint(row)
//		id ,ok := row["uid"]
//		if !ok {
//			return data,myerr.NewErrorCode(456)
//		}
//
//		pid := int(id.(float64))
//		playerListStruct = append(playerListStruct,Player{Id:pid})
//	}
//
//	addition := ""
//	additionStr,ok := jsonDataMap["addition"]
//	if ok {
//		addition = additionStr.(string)
//	}
//	data.GroupId = groupId
//	data.PlayersList = playerListStruct
//	data.CustomProp = customProp
//	data.RuleId = ruleId
//	data.Addition = addition
//	return data,nil
//}
