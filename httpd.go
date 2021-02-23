package gamematch

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"zlib"
)

//content_type 类型 枚举
const (
	CT_EMPTY 		= ""
	CT_JSON 		= "application/json"
	CT_MULTIPART 	= "multipart/form-data"
	CT_URLENCODED 	= "application/x-www-form-urlencoded"
	CT_STREAM 		= "application/octet-stream"
	CT_PLAIN 		= "text/plain"
	CT_HTML 		= "text/html"
	CT_JS 			= "application/javascript"
	CT_XML 			= "application/xml"
)
//请求-ContentType
type ContentType struct {
	Name		string
	Char 		string
	Addition	string
}
//响应结构体
type ResponseMsgST struct {
	Code 	int
	Msg 	interface{}
}
//httpd 类
type Httpd struct {
	Option HttpdOption
	Gamematch *Gamematch
	Log *zlib.Log
}
//实例化 初始值
type HttpdOption struct {
	Host	string
	Port 	string
	Log 	*zlib.Log
}
//实例化
func NewHttpd(httpdOption HttpdOption,gamematch *Gamematch) *Httpd{
	httpdOption.Log.Info("NewHttpd : ",httpdOption)
	httpd := new (Httpd)
	httpd.Option = httpdOption
	httpd.Gamematch = gamematch
	httpd.Log = getModuleLogInc("httpd")
	return httpd
}
//开启HTTP监听
func (httpd *Httpd)Start()error{
	//监听根目录uri
	http.HandleFunc("/", httpd.RouterHandler)
	dns := httpd.Option.Host + ":" + httpd.Option.Port
	httpd.Option.Log.Info("httpd start loop:",dns , " Listen : /")
	httpd.Log.Info("httpd start loop:",dns , " Listen : /")

	err := http.ListenAndServe(dns, nil)
	if err != nil {
		return errors.New("http.ListenAndServe() err :  " + err.Error())
	}
	return nil
}
//主要，是接收HTTP 回调
func (httpd *Httpd)RouterHandler(w http.ResponseWriter, r *http.Request){
	parameter := r.URL.Query()//GET 方式URL 中的参数 转 结构体
	uri := r.URL.RequestURI()

	contentType := httpd.GetContentType(r)
	//zlib.MyPrint("r.form",r.Form)
	httpd.Option.Log.Info("receiver :  uri :",uri," , url.query : ",parameter, " method : ",r.Method , " content_type : ",contentType)
	httpd.Log.Info("receiver :  uri :",uri," , url.query : ",parameter, " method : ",r.Method , " content_type : ",contentType)
	httpd.Log.Debug(r.Header)
	var postDataMap map[string]interface{}
	var postJsonStr string
	if strings.ToUpper(r.Method)  == "POST"{
		GetPostDataMap ,myJsonStr,errs := httpd.GetPostData(r,contentType.Name)
		//httpd.Option.Log.Debug("postDataMap : ",postDataMap)
		if errs != nil{
			httpd.ResponseStatusCode(w,500,"httpd.GetPostDat" + errs.Error() )
			return
		}
		httpd.Option.Log.Info(GetPostDataMap,errs)
		httpd.Log.Info(GetPostDataMap,errs)
		postDataMap = GetPostDataMap
		postJsonStr = myJsonStr
	}else{
		//this is get method
	}
	//time.Now().Format("2006-01-02 15:04:05")
	if r.URL.RequestURI() == "/favicon.ico" {//浏览器的ICON
		httpd.ResponseStatusCode(w,403,"no power")
		return
	}
	if uri == "" || uri == "/" {
		httpd.ResponseStatusCode(w,500,"RequestURI is null or uir is :  '/'")
		return
	}
	//去掉 URI 中最后一个 /
	uriLen := len(uri)
	if string([]byte(uri)[uriLen-1:uriLen]) == "/"{
		uri = string([]byte(uri)[0:uriLen - 1])
	}
	httpd.Log.Info("final uri : ",uri , " start routing ...")
	//*********: 还没有加  v1  v2 版本号
	code := 200
	var msg interface{}
	if uri == "/sign" {
		code,msg = httpd.signHandler(postJsonStr)
	}else if uri == "/sign/cancel"{
		code,msg = httpd.signCancelHandler(postJsonStr)
	}else if uri == "/rule/add" {
		//code,msg = httpd.ruleAddOne(postDataMap)
	}else if uri == "/getErrorInfo" {
		code,msg = httpd.getErrorInfoHandler()
	}else if uri == "/clearRuleByCode"{
		code,msg = httpd.clearRuleByCodeHandler(postDataMap)
	}else{
		code = 500
		msg = " uri router failed."
	}

	httpd.ResponseMsg(w,code,msg)

}
//响应的具体内容
func  (httpd *Httpd)ResponseMsg(w http.ResponseWriter,code int ,msg interface{} ){
	//fmt.Println("SetResponseMsg in",code,msg)
	responseMsgST := ResponseMsgST{Code:code,Msg:msg}
	//msg = msg[1:len(msg)-1]
	//这里有个无奈的地方，为了兼容非网络请求，正常使用时，返回的就是json,现在HTTP套一层，还得再一层JSON，冲突了
	//fmt.Println("responseMsg : ",responseMsg)
	jsonResponseMsg , err := json.Marshal(responseMsgST)
	//zlib.MyPrint(string(msg))
	//jsonResponseMsgNew := strings.Replace(string(jsonResponseMsg),"#msg#",msg,-1)
	if code == 200{
		httpd.Option.Log.Info("ResponseMsg rs",err, string(jsonResponseMsg))
		httpd.Log.Info("ResponseMsg",string(jsonResponseMsg))
	}else{
		httpd.Option.Log.Error("ResponseMsg rs",err, string(jsonResponseMsg))
		httpd.Log.Error("ResponseMsg",string(jsonResponseMsg))
	}

	w.Header().Set("Content-Length",strconv.Itoa( len(jsonResponseMsg) ) )
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(jsonResponseMsg))
}
//http 响应状态码
func (httpd *Httpd)ResponseStatusCode(w http.ResponseWriter,code int ,responseInfo string){
	httpd.Option.Log.Info("ResponseStatusCode",code,responseInfo)
	httpd.Log.Info("ResponseStatusCode",code,responseInfo)

	w.Header().Set("Content-Length",strconv.Itoa( len(responseInfo) ) )
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(403)
	w.Write([]byte(responseInfo))
}


func (httpd *Httpd) GetContentType( r *http.Request)ContentType{
	//r.Header.Get("Content-Type")
	contentTypeArr ,ok := r.Header["Content-Type"]
	httpd.Option.Log.Debug(contentTypeArr)

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
	list := []string{
		CT_JSON,CT_MULTIPART,CT_URLENCODED,CT_STREAM,CT_EMPTY,CT_PLAIN,CT_JS,CT_HTML,CT_XML,
	}
	return list
}

func (httpd *Httpd)GetPostData(r *http.Request,contentType string)( data  map[string]interface{},jsonStr string, err error){
	httpd.Option.Log.Debug(" getPostData ")
	if r.ContentLength == 0{//获取主体数据的长度
		return data,jsonStr,nil
	}
	switch contentType {
		case CT_JSON:
			body := make([]byte, r.ContentLength)
			r.Body.Read(body)
			mylog.Debug("body : ",string(body))

			jsonDataMap := make(map[string]interface{})
			errs := json.Unmarshal(body,&jsonDataMap)
			mylog.Debug("test ",jsonDataMap,errs)
			return jsonDataMap,string(body),nil
		case CT_MULTIPART:
			data = make( map[string]interface{})
			r.ParseMultipartForm(r.ContentLength)
			for k,v:= range r.Form{
				data[k] = v
			}
			return data,jsonStr,nil
		case CT_URLENCODED:
			err := r.ParseForm()
			if err != nil{
				return data,jsonStr,err
			}
			data = make( map[string]interface{})
			for k,v:= range r.Form{
				if len(v) == 1{
					data[k] = v[0]
				}else{
					zlib.ExitPrint(" bug !!!")
				}
			}
			return data,jsonStr,nil
		default:
			httpd.Option.Log.Error("contentType no support : ",contentType , " ,no data")
	}

	return data,jsonStr,nil
}