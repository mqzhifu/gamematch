package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"src/zlib"
)

func NewRedisConn(redisHost string , redisPort string)(redis.Conn,error){
	log.Info("NewRedisConn : ",redisHost,redisPort)
	conn,error := redis.Dial("tcp",redisHost+":"+redisPort)
	if error != nil{
		return nil,err.NewErrorCode(300)
	}
	return conn,nil
}

func redisDo(commandName string, args ...interface{})(reply interface{}, error error){
	log.Debug("[redis]redisDo init:",commandName,args)
	res,error := redisConn.Do(commandName,args... )
	if error != nil{
		errInfo := make(map[int]string)
		errInfo[0] = error.Error()
		return nil,err.NewErrorCodeReplace(301,errInfo)
	}
	//reflect.ValueOf(res).IsNil(),reflect.ValueOf(res).Kind(),reflect.TypeOf(res)
	//zlib.MyPrint("redisDo exec ,res : ",res," err :",err)
	return res,error
}

func redisDelAllByPrefix(prefix string){
	log.Notice(" action redisDelAllByPrefix : ",prefix)
	res,err := redis.Strings(  redisDo("keys",prefix))
	if err != nil{
		zlib.ExitPrint("redis keys err :",err.Error())
	}
	log.Debug("del element will num :",len(res))
	if len(res) <= 0{
		log.Notice(" keys is null,no need del...")
		return
	}
	for _,v := range res{
		res,_ :=redisDo("del",v)
		log.Debug("del key ",v , " ,  rs : ",res)
	}
}

func redisDelAll(){
	redisDelAllByPrefix(redisPrefix)
}