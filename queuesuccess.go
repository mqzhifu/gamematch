package gamematch

type QueueSuccessElement struct {
	Id 			int
	ATime		int		//匹配成功的时间
	Timeout		int		//多少秒后无人来取，即超时，更新用户状态，删除数据
	TeamTotal	int		//该结果，有几个 队伍，因为每个队伍是一个集合，要用来索引
}

type QueueSuccessPlayerElements struct {
	PlayerId	int
	TeamId		int
	TeamWeight	int
	GroupId		int
	GroupWeight	int
}

type QueueSuccess struct {

}

func NewQueueSuccess()*QueueSuccess{
	queueSuccess := new(QueueSuccess)
	return queueSuccess
}
////获取一个自增ID
//func (queueSuccess *QueueSuccess) getAutoIncrementId(ruleId int)int {
//	key := redisPrefix + redisSeparation + "suce" + redisSeparation + "inc" + redisSeparation + strconv.Itoa(ruleId)
//	res,err := redis.Int(redisDo("INCR",key))
//	if err != nil{
//		zlib.ExitPrint("sucess getAutoIncrementId err ",err.Error())
//	}
//	return res
//}
//
//func (queueSuccess *QueueSuccess) getRedisKey(ruleId int)string{
//	return redisPrefix + redisSeparation+ "suce" + redisSeparation + strconv.Itoa(ruleId)
//}
//
//func (queueSuccess *QueueSuccess) getRedisPlayerKey(pid int)string{
//	return redisPrefix + redisSeparation+ "suce" + redisSeparation + "player" +redisSeparation + strconv.Itoa(pid)
//}
//
//func (queueSuccess *QueueSuccess) NewQueueSuccess(){
//
//}
//
//func (queueSuccess *QueueSuccess) PushOne(successPlayers map[int]QueueSignElement,rule Rule){
//	playerStructs := make(map[int]string)
//	i := 0
//	for _,v := range successPlayers{
//		queueSuccessPlayerElements := QueueSuccessPlayerElements{
//			PlayerId: v.PlayerId,
//			TeamId: 0,
//			GroupId: v.GroupId,
//			TeamWeight:0,
//		}
//		queueSuccessPlayerElementsStr := queueSuccess.structToStrPlayer(queueSuccessPlayerElements)
//		playerStructs[i] = queueSuccessPlayerElementsStr
//		i++
//	}
//	id := queueSuccess.getAutoIncrementId(rule.Id)
//	queueSuccessElement := QueueSuccessElement{
//		Id :id,
//		ATime: getNowTime(),
//		Timeout: rule.SuccessTimeout,
//	}
//	queueSuccessElementStr := queueSuccess.structToStr(queueSuccessElement)
//	key := queueSuccess.getRedisKey
//	redisDo("ZADD",redis.Args{}.Add(key).Add(queueSuccessElement.ATime).Add(queueSuccessElementStr)...)
//	playerKey := queueSuccess.getRedisPlayerKey(id)
//	for _,v := range playerStructs{
//		zlib.MyPrint(v)
//		redisDo("set",redis.Args{}.Add(playerKey).Add())
//	}
//
//}
//
//func (queueSuccess *QueueSuccess)  strToStruct(redisStr string )QueueSuccessElement{
//	strArr := strings.Split(redisStr,separation)
//	element := QueueSuccessElement{
//		Id:zlib.Atoi(strArr[0]),
//		Timeout 	:	zlib.Atoi(strArr[1]),
//		ATime 	:	zlib.Atoi(strArr[2]),
//	}
//	return element
//}
//
//func (queueSuccess *QueueSuccess) structToStr(queueSuccessElement QueueSuccessElement)string{
//	content :=
//		strconv.Itoa( queueSuccessElement.Id) + separation +
//			strconv.Itoa( queueSuccessElement.Timeout) + separation +
//			strconv.Itoa( queueSuccessElement.ATime)
//	return content
//}
//
//func (queueSuccess *QueueSuccess)  strToStructPlayer(redisStr string )QueueSuccessPlayerElements{
//	strArr := strings.Split(redisStr,separation)
//	element := QueueSuccessPlayerElements{
//		PlayerId 	:	zlib.Atoi(strArr[0]),
//		GroupId 	:	zlib.Atoi(strArr[1]),
//		TeamId 		:	zlib.Atoi(strArr[2]),
//		TeamWeight 	:	zlib.Atoi(strArr[3]),
//	}
//	return element
//}
//
//func (queueSuccess *QueueSuccess) structToStrPlayer(queueSuccessPlayerElements QueueSuccessPlayerElements)string{
//	content :=
//			strconv.Itoa( queueSuccessPlayerElements.PlayerId) + separation +
//			strconv.Itoa( queueSuccessPlayerElements.GroupId) + separation +
//			strconv.Itoa( queueSuccessPlayerElements.TeamId )+ separation +
//			strconv.Itoa( queueSuccessPlayerElements.TeamWeight)+ separation
//	return content
//}
//
//func (queueSuccess *QueueSuccess) popOne(){
//
//}
//
//func (queueSuccess *QueueSuccess) getList(){
//
//}
