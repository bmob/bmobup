package main

import (
	"bmob/library/file"
	"bmob/library/log"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Unknwon/goconfig"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type App struct {
	status    int
	url       string
	appPath   string //函数路径
	funcName  string //函数名
	id        string
	key       string
	secretKey string
	language  int
	log       *log.Logger
}

func main() {

	app := new(App)

	// 获取Key
	if ok := app.init(); !ok {
		app.log.Error("初始化失败")
		return
	}

	//获取代码内容
	params := os.Args
	if len(params) < 2 {
		app.log.Error("必须输入文件名称")
		log.Error("必须输入文件名称")
		return
	}

	// 读取代码内容
	fileName := params[1]
	code, ok := app.readFile(fileName)
	if !ok {
		app.log.Error("代码读取错误")
		return
	}
	_, ok = app.sendCode(code)
	if !ok {
		app.log.Error("代码更新失败")
		return
	}

	result, ok := app.viewCloud()
	if !ok {
		app.log.Error("代码更新失败")
		return
	}

	fmt.Printf("结果%s\n", result)

}

func (app *App) request(method string, url string, row string) (string, bool) {

	client := &http.Client{}
	res, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(row)))

	res.Header.Set("Content-Type", "application/json")
	if app.status == 1 {
		res.Header.Set("X-Bmob-Application-Id", app.id)
		res.Header.Set("X-Bmob-REST-API-Key", app.key)
	}
	if err != nil {
		app.log.Error("Fatal error ", err.Error())
	}

	//处理返回结果
	response, err := client.Do(res)
	if err != nil {
		app.log.Error("Fatal error ", err.Error())
	}

	//返回的状态码
	status := response.StatusCode
	b, err := ioutil.ReadAll(response.Body)
	if err != nil || status != 200 {
		// handle error
		app.log.Error("%s,%s,更新错误原因：%s", url, method, string(b))
		fmt.Printf("错误返回：%s \n 具体请查看log日志", string(b))
		return "", false
	}

	return string(b), true

}

// 预览云函数结果
func (app *App) viewCloud() (string, bool) {
	app.status = 2
	fmt.Printf("云端结果返回中...\n")
	url := "https://cloud.bmob.cn/" + app.secretKey + "/" + app.funcName
	if app.language == 2 {
		url = "https://javacloud.bmob.cn/" + app.secretKey + "/" + app.funcName
	}
	fmt.Println("url", url)
	result, ok := app.request("GET", url, "")

	if !ok {
		app.log.Error("请求错误")
		return "", false
	}
	return result, true

}

//发送代码到服务器
func (app *App) sendCode(code string) (string, bool) {
	fmt.Printf("云代码" + app.funcName + "上传中...\n")
	url := "https://api2.bmob.cn" + "/1/functions/" + app.funcName

	type bodyt struct {
		Code     string `json:"code"`
		Language int    `json:"language"`
		Comment  string `json:"comment"`
	}

	app.status = 1

	// 编码
	encStr := base64.StdEncoding.EncodeToString([]byte(code))

	var body bodyt
	body.Language = app.language
	body.Code = encStr
	body.Comment = ""

	app.log.Debug("发送结构体：%s", body)

	b, err := json.Marshal(body)
	if err != nil {
		app.log.Error("json err:", err)
	}
	row := string(b)
	result, ok := app.request("PUT", url, row)

	if !ok {
		app.log.Error("请求错误")
		return "", false
	}
	return result, true
}

//返回文件路径
func (app *App) fileExists(fileName string) (string, bool) {
	configFile := filepath.Join(app.appPath, "", fileName)
	if !file.FileExists(configFile) {
		app.log.Error("读取文件错误")
		return "", false
	}
	return configFile, true
}

// 读取代码文件内容
func (app *App) readFile(fileName string) (string, bool) {
	nameArr := strings.Split(fileName, ".")
	app.funcName = nameArr[0]

	language := 9
	if nameArr[1] == "js" {
		language = 1
	}
	if nameArr[1] == "java" {
		language = 2
	}

	if language == 9 {
		app.log.Error("云函数文件格式错误,%s", nameArr[1])
		fmt.Printf("云函数文件格式错误,%s \n", nameArr[1])
		return "", false
	}

	app.language = language
	configFile, ok := app.fileExists(fileName)
	if !ok {
		app.log.Error("目录文件不存在")
		return "", false
	}

	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		app.log.Error("读取文件错误")
		return "", false
	}

	content := string(bytes)
	return content, true
}

func (app *App) init() bool {
	appPath := file.GetCurrentPath()

	app.appPath = appPath

	//日志配置
	appLog := new(log.Logger)
	var err error
	appLog, err = log.New(
		"info",
		"log/all.log",
		7)
	if err != nil {
		panic(err.Error())
	}
	app.log = appLog

	//配置文件
	configFile, ok := app.fileExists("config.ini")

	if !ok {
		app.log.Error("目录文件不存在")
		return false
	}

	C, err := goconfig.LoadConfigFile(configFile)
	if err != nil {
		app.log.Error("load config file error %s", err)
		return false
	}

	app.id = C.MustValue("app", "id", "")
	app.key = C.MustValue("app", "key", "")
	app.secretKey = C.MustValue("app", "secretKey", "")
	return true
}
