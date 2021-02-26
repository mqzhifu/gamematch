package gamematch

import (
	"encoding/json"
	"strings"
	"zlib"
)

func  (httpd *Httpd)getConstList(){
	httpd.Log.Info(" routing in signHandler : ")

	fileDir := "/data/www/golang/src/gamematch/const.go"
	doc := zlib.NewDocRegular(fileDir)
	doc.ParseConst()
}
func  (httpd *Httpd)checkHttpdState(ruleId int)(bool,error){
	state ,ok := httpd.Gamematch.HttpdRuleState[ruleId]
	if !ok {
		return false,myerr.NewErrorCode(803)
	}
	if state == HTTPD_RULE_STATE_OK{
		return true,nil
	}
	return false,myerr.NewErrorCode(804)
}
//获取错误码
func  (httpd *Httpd)getErrorInfoHandler()(code int ,msg interface{}){
	httpd.Log.Info(" routing in getErrorInfoHandler : ")

	container := getErrorCode()
	var res []MyErrorCode
	for _,v:= range  container{
		row := strings.Split(v,",")
		myErrorCode := MyErrorCode{
			Code: zlib.Atoi(row[0]),
			Msg :row[1],
			Flag: row[2],
			MsgCn: row[3],
		}
		res = append(res,myErrorCode)
	}

	//msg,_ := json.Marshal(res)
	//zlib.MyPrint(string(msg))
	return 200,res
}
func  (httpd *Httpd)clearRuleByCodeHandler(postDataMap map[string]interface{})(code int ,msg interface{}){
	httpd.Log.Info(" routing in clearRuleByCodeHandler : ")
	if len(postDataMap) == 0{
		errs := myerr.NewErrorCode(802)
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)
		return errInfo.Code,errInfo.Msg
	}

	matchCode,ok  := postDataMap["matchCode"]
	//httpd.Option.Log.Info(matchCode,ok)
	if !ok || matchCode == ""{
		errs := myerr.NewErrorCode(450)
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)

		return errInfo.Code,errInfo.Msg
	}

	checkCodeRs := false
	ruleId := 0
	//zlib.ExitPrint(httpd.Gamematch.RuleConfig.getAll())
	for _,rule := range httpd.Gamematch.RuleConfig.getAll(){
		if  rule.CategoryKey == matchCode{
			ruleId = rule.Id
			checkCodeRs = true
			break
		}
	}
	if !checkCodeRs{
		errs := myerr.NewErrorCode(451)
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)

		return errInfo.Code,errInfo.Msg
	}

	httpd.Gamematch.delOneRule(ruleId)
	return code,msg
}

//报名 - 添加匹配玩家
func  (httpd *Httpd)signHandler( postJsonStr string)(code int ,msg interface{}){
	httpd.Log.Info(" routing in signHandler : ")
	if postJsonStr == ""{
		errs := myerr.NewErrorCode(802)
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)
		return errInfo.Code,errInfo.Msg
	}
	httpReqSign := HttpReqSign{}
	err := json.Unmarshal([]byte(postJsonStr),&httpReqSign)
	if err != nil{
		zlib.ExitPrint("un json str err",err.Error())
	}
	newHttpReqSign,errs := httpd.Gamematch.CheckHttpSignData(httpReqSign)
	//zlib.ExitPrint(httpReqSign)
	if errs != nil{
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)

		return errInfo.Code,errInfo.Msg
	}
	_,err  = httpd.checkHttpdState(newHttpReqSign.RuleId)
	if err != nil{
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)

		return errInfo.Code,errInfo.Msg
	}
	//signRsData, errs := httpd.Gamematch.Sign(data.RuleId,data.GroupId,data.CustomProp,data.PlayersList,data.Addition)
	signRsData, errs := httpd.Gamematch.Sign(newHttpReqSign)
	if errs != nil{
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)

		return errInfo.Code,errInfo.Msg
	}
	return 200,signRsData
}
//取消报名 - 删除已参与匹配的玩家信息
func  (httpd *Httpd)signCancelHandler(postJsonStr string)(code int ,msg interface{}){
	httpd.Log.Info(" routing in signCancelHandler : ")
	if postJsonStr == ""{
		errs := myerr.NewErrorCode(802)
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)
		return errInfo.Code,errInfo.Msg
	}
	httpReqSignCancel := HttpReqSignCancel{}
	err := json.Unmarshal([]byte(postJsonStr),&httpReqSignCancel)
	if err != nil{
		zlib.ExitPrint("un json str err",err.Error())
	}

	//jsonDataMap := make(map[string]string)
	//jsonDataMap["groupId"] = zlib.Float64ToString(postDataMap["groupId"].(float64),0)
	//jsonDataMap["matchCode"] = postDataMap["matchCode"].(string)
	data,errs := httpd.Gamematch.CheckHttpSignCancelData(httpReqSignCancel)
	if errs != nil{
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)
		return errInfo.Code,  errInfo.Msg
	}

	_,err  = httpd.checkHttpdState(data.RuleId)
	if err != nil{
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)

		return errInfo.Code,errInfo.Msg
	}

	signClass := httpd.Gamematch.getContainerSignByRuleId(data.RuleId)
	if data.GroupId > 0{
		httpd.Log.Info("del by groupId")
		err := signClass.cancelByGroupId(data.GroupId)
		if err != nil{
			errInfo := zlib.ErrInfo{}
			json.Unmarshal([]byte(err.Error()),&errInfo)

			return errInfo.Code,errInfo.Msg
		}
	}else{
		httpd.Log.Info("del by playerId")
		//	signClass.cancelByPlayerId(data.PlayerId)
	}

	return 200,"ok"
}



//----------------------------------------------------------------------

//func  (httpd *Httpd)ruleAddOne(  postDataMap map[string]interface{})(code int ,msg interface{}){
//	data,errs := httpd.Gamematch.CheckHttpAddRule(jsonDataMap)
//	//zlib.MyPrint(errs)
//	if errs != nil{
//		errInfo := zlib.ErrInfo{}
//		json.Unmarshal([]byte(errs.Error()),&errInfo)
//
//		return errInfo.Code,errInfo.Msg
//	}
//
//	httpd.Gamematch.RuleConfig.AddOne()
//
//	return code,msg
//}