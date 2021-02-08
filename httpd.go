package gamematch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"src/zlib"
	"strconv"
	"strings"
	"time"
)

type ResponseMsgST struct {
	Code 	int
	Msg 	interface{}
}

type Httpd struct {
	Port 		int
	Host 		string
	Gamematch 	*Gamematch
}

func NewHttpd(port int ,host string) *Httpd{
	zlib.MyPrint("NewHttpd : ",host,port)
	httpd := new (Httpd)
	httpd.Port = port
	httpd.Host = host
	//gamematch,errs := NewGamematch()
	//if errs != nil{
	//	zlib.ExitPrint("NewGamematch err  :",errs.Error())
	//}
	//httpd.Gamematch = gamematch
	//httpd.Gamematch.testSignDel(2)
	return httpd
}

func (httpd *Httpd)Start(){

	http.HandleFunc("/", httpd.RouterHandler)
	dns := httpd.Host + ":" + strconv.Itoa(httpd.Port)
	zlib.MyPrint("httpd start loop:",dns)

	err := http.ListenAndServe(dns, nil)
	if err != nil {
		fmt.Printf("http.ListenAndServe()函数执行错误,错误为:%v\n", err)
		return
	}
}
func (httpd *Httpd)getPostData(w http.ResponseWriter,r *http.Request)(  map[string]interface{}, error){
	//if r.ContentLength == 0{//获取主体数据的长度
	//	errs := err.NewErrorCode(800)
	//	errInfo := zlib.ErrInfo{}
	//	json.Unmarshal([]byte(errs.Error()),&errInfo)
	//	ResponseMsg(w,errInfo.Code,  errInfo.Msg)
	//	return nil,errs
	//}
	body := make([]byte, r.ContentLength)
	r.Body.Read(body)
	mylog.Debug("body : ",string(body))
	jsonDataMap := make(map[string]interface{})
	errs := json.Unmarshal(body,&jsonDataMap)
	mylog.Debug("test ",jsonDataMap,errs)
	return jsonDataMap,nil

}
func ResponseMsg(w http.ResponseWriter,code int ,msg interface{} ){
	//fmt.Println("SetResponseMsg in",code,msg)
	//responseMsgST := ResponseMsgST{Code:code,Msg:msg}
	responseMsgST := ResponseMsgST{Code:code,Msg:msg}
	//msg = msg[1:len(msg)-1]
	//这里有个无奈的地方，为了兼容非网络请求，正常使用时，返回的就是json,现在HTTP套一层，还得再一层JSON，冲突了
	//fmt.Println("responseMsg : ",responseMsg)
	jsonResponseMsg , err := json.Marshal(responseMsgST)
	//zlib.MyPrint(string(msg))
	//jsonResponseMsgNew := strings.Replace(string(jsonResponseMsg),"#msg#",msg,-1)
	fmt.Println("SetResponseMsg rs",err, string(jsonResponseMsg))

	_, _ = w.Write([]byte(jsonResponseMsg))

}

func ResponseStatusCode(w http.ResponseWriter,code int ,responseInfo string){
	w.Header().Set("Content-Length",strconv.Itoa( len(responseInfo) ) )
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(403)
	w.Write([]byte(responseInfo))
}
//主要，是接收HTTP 回调
func (httpd *Httpd)RouterHandler(w http.ResponseWriter, r *http.Request){
	//fmt.Printf("时间戳（秒）：%v;\n", time.Now().Unix())
	//time.Now().UnixNano()

	zlib.MyPrint("receiver: have a new request.(", time.Now().Format("2006-01-02 15:04:05"),")")
	parameter := r.URL.Query()
	zlib.MyPrint("uri",r.URL.RequestURI(),"url.query",parameter)
	if r.URL.RequestURI() == "/favicon.ico" {
		ResponseStatusCode(w,403,"no power")
		return
	}
	if r.URL.RequestURI() == "" || r.URL.RequestURI() == "/" {
		ResponseStatusCode(w,500,"RequestURI is null or uir is :  '/'")
		return
	}


	//*********: 还没有加  v1  v2 版本号

	if r.URL.RequestURI() == "/sign/" || r.URL.RequestURI() == "/sign" {
		jsonDataMap ,errs := httpd.getPostData(w,r)
		if errs != nil{
			return
		}
		data,errs := httpd.Gamematch.CheckHttpSignData(jsonDataMap)
		//zlib.MyPrint(errs)
		if errs != nil{
			errInfo := zlib.ErrInfo{}
			json.Unmarshal([]byte(errs.Error()),&errInfo)

			ResponseMsg(w,errInfo.Code,  errInfo.Msg)
			return
		}

		errs = httpd.Gamematch.Sign(data.RuleId,data.GroupId,data.CustomProp,data.PlayersList,data.Addition)
		if errs != nil{
			errInfo := zlib.ErrInfo{}
			json.Unmarshal([]byte(errs.Error()),&errInfo)

			ResponseMsg(w,errInfo.Code,  errInfo.Msg)
			return
		}
		ResponseMsg(w,200,"")
		return
	}else if r.URL.RequestURI() == "/sign/cancel" || r.URL.RequestURI() == "/sign/cancel/" {
		jsonDataMapTmp,errs := httpd.getPostData(w,r)
		if errs != nil{
			return
		}
		jsonDataMap := make(map[string]string)
		for k,v :=range jsonDataMapTmp{
			jsonDataMap[k] = v.(string)
		}
		data,errs := httpd.Gamematch.CheckHttpSignCancelData(jsonDataMap)
		zlib.MyPrint(errs)
		if errs != nil{
			errInfo := zlib.ErrInfo{}
			json.Unmarshal([]byte(errs.Error()),&errInfo)

			ResponseMsg(w,errInfo.Code,  errInfo.Msg)
			return
		}

		signClass := httpd.Gamematch.getContainerSignByRuleId(data.RuleId)
		signClass.cancel(data.PlayerId)
	}else if r.URL.RequestURI() == "/getErrorInfo" || r.URL.RequestURI() == "/getErrorInfo/" {
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
		ResponseMsg(w,200, res )
		return
	}else{
		//errs := err.NewErrorCode(801)
		//errInfo := zlib.ErrInfo{}
		//json.Unmarshal([]byte(errs.Error()),&errInfo)
		//ResponseMsg(w,errInfo.Code,  errInfo.Msg)
		return
	}

	//zlib.MyPrint("searchRs",searchRs)
	//ResponseMsg(w,200,searchRs)

}


