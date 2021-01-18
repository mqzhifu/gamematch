package gamematch

import (
	"github.com/gomodule/redigo/redis"
	"src/zlib"
)

func InitConn(redisHost string , redisPort string)redis.Conn{

	conn,err := redis.Dial("tcp",redisHost+":"+redisPort)
	if err != nil{
		zlib.ExitPrint("redis conn err:",err)
	}
	return conn
}

func redisDo(commandName string, args ...interface{})(reply interface{}, err error){
	zlib.MyPrint("redisDo init:",commandName,args)
	res,err := redisConn.Do(commandName,args... )
	//reflect.ValueOf(res).IsNil(),reflect.ValueOf(res).Kind(),reflect.TypeOf(res)
	zlib.MyPrint("redisDo exec ,res : ",res," err :",err)
	return res,err
}

func redisDelAllByPrefix(prefix string){
	res,err := redis.Strings(  redisDo("keys",prefix))
	if err != nil{
		zlib.ExitPrint("redis keys err :",err.Error())
	}
	zlib.MyPrint("del element will num :",len(res))
	if len(res) <= 0{
		zlib.MyPrint(" keys is null,no need del...")
		return
	}
	for _,v := range res{
		redisDo("del",v)
	}
}

func redisDelAll(){
	redisDelAllByPrefix(redisPrefix)
}