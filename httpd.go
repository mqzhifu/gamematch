package gamematch

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"src/zlib"
	"strconv"
	"strings"
)

const (
	CT_JSON = "application/json"
	CT_MULTIPART = "multipart/form-data"
	CT_URLENCODED = "application/x-www-form-urlencoded"
	CT_STREAM = "application/octet-stream"
	CT_EMPTY = ""
)

type ContentType struct {
	Name		string
	Char 		string
	Addition	string
}

type ResponseMsgST struct {
	Code 	int
	Msg 	interface{}
}

type Httpd struct {
	Option HttpdOption
	Gamematch *Gamematch
}

type HttpdOption struct {
	Host	string
	Port 	string
	Log 	*zlib.Log

}

func NewHttpd(httpdOption HttpdOption,gamematch *Gamematch) *Httpd{
	httpdOption.Log.Info("NewHttpd : ",httpdOption)
	httpd := new (Httpd)
	httpd.Option = httpdOption
	httpd.Gamematch = gamematch
	return httpd
}

func (httpd *Httpd)Start()error{
	//监听根目录uri
	http.HandleFunc("/", httpd.RouterHandler)
	dns := httpd.Option.Host + ":" + httpd.Option.Port
	httpd.Option.Log.Info("httpd start loop:",dns , " Listen : /")

	err := http.ListenAndServe(dns, nil)
	if err != nil {
		return errors.New("http.ListenAndServe() err :  " + err.Error())
	}
	return nil
}



func ResponseMsg(w http.ResponseWriter,code int ,msg interface{} ){
	//fmt.Println("SetResponseMsg in",code,msg)
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
	//zlib.ExitPrint(r.Header)
	parameter := r.URL.Query()//GET 方式URL 中的参数 转 结构体
	uri := r.URL.RequestURI()

	contentType := httpd.GetContentType(r)
	//zlib.MyPrint("r.form",r.Form)
	httpd.Option.Log.Info("receiver :  uri :",uri," , url.query : ",parameter, " method : ",r.Method , " content_type : ",contentType)
	//zlib.ExitPrint(r.Header)
	var postDataMap map[string]interface{}
	if strings.ToUpper(r.Method)  == "POST"{
		postDataMap ,errs := httpd.GetPostData(r,contentType.Name)
		if errs != nil{
		 	ResponseStatusCode(w,500,"httpd.GetPostDat" + errs.Error() )
			return
		}
		httpd.Option.Log.Info(" post data : ",postDataMap)
	}else{

	}
	//time.Now().Format("2006-01-02 15:04:05")
	if r.URL.RequestURI() == "/favicon.ico" {//浏览器的ICON
		ResponseStatusCode(w,403,"no power")
		return
	}
	if uri == "" || uri == "/" {
		ResponseStatusCode(w,500,"RequestURI is null or uir is :  '/'")
		return
	}
	//去掉 URI 中最后一个 /
	uriLen := len(uri)
	if string([]byte(uri)[0:uriLen - 1]) == "/"{
		uri = string([]byte(uri)[0:uriLen - 2])
	}

	//*********: 还没有加  v1  v2 版本号
	code := 200
	var msg interface{}
	if uri == "/sign" {
		code,msg = httpd.signHandler(postDataMap)
	}else if r.URL.RequestURI() == "/sign/cancel"{
		code,msg = httpd.signCancelHandler(postDataMap)
	}else if r.URL.RequestURI() == "/rule/add" {
		//code,msg = httpd.ruleAddOne(postDataMap)
	}else if r.URL.RequestURI() == "/getErrorInfo" {
		code,msg = httpd.getErrorInfo()
	}else{
		code = 500
		msg = " uri router failed."
	}

	ResponseMsg(w,code,msg)

}

func (httpd *Httpd) GetContentType( r *http.Request)ContentType{
	//r.Header.Get("Content-Type")
	contentTypeArr ,ok := r.Header["Content-Type"]
	//httpd.Option.Log.Debug(contentTypeArr)

	//正常的请求基本上没这个值，除了 FORM，因为只有传输内容的时候才有意义
	contentType := ContentType{}
	if ok {
		contentType.Name = contentTypeArr[0]
		httpd.Option.Log.Debug(contentType.Name)
		if strings.Index(contentType.Name,"multipart/form-data") != -1{
			tmpArr := strings.Split(contentType.Name,";")
			contentType.Addition = strings.TrimSpace(tmpArr[1])
			contentType.Name = CT_MULTIPART
		}else{
			tmpArr := strings.Split(contentType.Name,";")
			if len(tmpArr) >= 2{
				contentType.Char = strings.TrimSpace(tmpArr[1])
			}
		}
		elementIndex := zlib.ElementStrInArrIndex(GetContentTypeList(),contentType.Name)
		if elementIndex == -1{
			httpd.Option.Log.Notice("content type is unknow ")
		}
	}else{
		contentType.Name =CT_EMPTY
	}
	return contentType
}


func GetContentTypeList()[]string{
	list := []string{CT_JSON,CT_MULTIPART,CT_URLENCODED,CT_STREAM,CT_EMPTY}
	return list
}

func (httpd *Httpd)GetPostData(r *http.Request,contentType string)( data  map[string]interface{}, err error){
	httpd.Option.Log.Debug(" getPostData ")
	if r.ContentLength == 0{//获取主体数据的长度
		return data,nil
	}
	switch contentType {
	case CT_JSON:
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		mylog.Debug("body : ",string(body))

		jsonDataMap := make(map[string]interface{})
		errs := json.Unmarshal(body,&jsonDataMap)
		mylog.Debug("test ",jsonDataMap,errs)
		return jsonDataMap,nil
	case CT_MULTIPART:
		r.ParseMultipartForm(r.ContentLength)
		for k,v:= range r.Form{
			data[k] = v
		}
		return data,nil
	//case "x-www-form-urlencoded":
	case CT_URLENCODED:
		err := r.ParseForm()
		if err != nil{
			return data,err
		}
		for k,v:= range r.Form{
			data[k] = v
		}
		return data,nil
	default:
		httpd.Option.Log.Error("contentType no support : ",contentType , " ,no data")
	}

	return data,nil
}


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

func getConstList(){
	fileDir := "/data/www/golang/src/gamematch/const.go"
	doc := zlib.NewDocRegular(fileDir)
	doc.ParseConst()
}


func  (httpd *Httpd)getErrorInfo()(code int ,msg interface{}){
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


func  (httpd *Httpd)signHandler(postDataMap map[string]interface{})(code int ,msg interface{}){
	data,errs := httpd.Gamematch.CheckHttpSignData(postDataMap)
	//zlib.MyPrint(errs)
	if errs != nil{
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)

		return errInfo.Code,errInfo.Msg
	}

	errs = httpd.Gamematch.Sign(data.RuleId,data.GroupId,data.CustomProp,data.PlayersList,data.Addition)
	if errs != nil{
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)

		return errInfo.Code,errInfo.Msg
	}
	return 200,""
}

func  (httpd *Httpd)signCancelHandler(postDataMap map[string]interface{})(code int ,msg interface{}){
	jsonDataMap := make(map[string]string)
	for k,v :=range postDataMap{
		jsonDataMap[k] = v.(string)
	}
	data,errs := httpd.Gamematch.CheckHttpSignCancelData(jsonDataMap)
	if errs != nil{
		errInfo := zlib.ErrInfo{}
		json.Unmarshal([]byte(errs.Error()),&errInfo)
		return errInfo.Code,  errInfo.Msg
	}

	signClass := httpd.Gamematch.getContainerSignByRuleId(data.RuleId)
	signClass.cancel(data.PlayerId)
	return 200,"ok"
}


