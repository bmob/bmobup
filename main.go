package main

import (
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
	"flag"
	"github.com/google/logger"
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
}

func main() {

	app := new(App)

	// 获取Key
	if ok := app.init(); !ok {
		logger.Errorf("初始化失败")
		return
	}

	//获取代码内容
	params := os.Args
	if len(params) < 2 {
		logger.Errorf("必须输入文件名称")
		return
	}

	// 读取代码内容
	fileName := params[1]
	code, ok := app.readFile(fileName)
	if !ok {
		logger.Errorf("代码读取错误")
		return
	}
	_, ok = app.sendCode(code)
	if !ok {
		logger.Errorf("代码更新失败")
		return
	}

	result, ok := app.viewCloud()
	if !ok {
		logger.Errorf("代码更新失败")
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
		logger.Errorf("Fatal error ", err.Error())
	}

	//处理返回结果
	response, err := client.Do(res)
	if err != nil {
		logger.Errorf("Fatal error ", err.Error())
	}

	//返回的状态码
	status := response.StatusCode
	b, err := ioutil.ReadAll(response.Body)
	if err != nil || status != 200 {
		// handle error
		logger.Errorf("%s,%s,更新错误原因：%s", url, method, string(b))
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
		logger.Errorf("请求错误")
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

	logger.Info("发送结构体：%s", body)

	b, err := json.Marshal(body)
	if err != nil {
		logger.Errorf("json err:", err)
	}
	row := string(b)
	result, ok := app.request("PUT", url, row)

	if !ok {
		logger.Errorf("请求错误")
		return "", false
	}
	return result, true
}

//返回文件路径
func (app *App) fileExists(fileName string) (string, bool) {
	configFile := filepath.Join(app.appPath, "", fileName)
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		logger.Errorf("配置文件 %s 不存在", configFile)
		return "", false
	}
	return configFile, true
}

// 读取代码文件内容
func (app *App) readFile(fileName string) (string, bool) {
	basename := filepath.Base(fileName)
	app.funcName = strings.TrimSuffix(basename, filepath.Ext(basename))
	if strings.HasSuffix(fileName, "js") {
		app.language = 1
	} else if strings.HasSuffix(fileName, "java") {
		app.language = 2
	} else {
		logger.Errorf("云函数文件格式错误：%s", fileName)
		fmt.Printf("云函数文件格式错误：%s \n", fileName)
		return "", false
	}
	
	dir, err := os.Getwd()
	if err != nil {
		logger.Fatal(err)
	}
	codeFile, err := filepath.Abs(filepath.Join(dir, "", fileName))
	if err != nil {
		logger.Errorf("目录文件不存在")
		return "", false
    }

	bytes, err := ioutil.ReadFile(codeFile)
	if err != nil {
		logger.Errorf("读取文件错误")
		return "", false
	}

	content := string(bytes)
	return content, true
}

func (app *App) init() bool {
	//日志配置
	const logPath = "bmobup.log"

	var verbose = flag.Bool("verbose", false, "print info level logs to stdout")
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
	logger.Fatalf("Failed to open log file: %v", err)
	}
	defer lf.Close()

	defer logger.Init("LoggerExample", *verbose, true, lf).Close()


	appPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
    if err != nil {
		logger.Fatal(err)
    }

	app.appPath = appPath

	//配置文件
	configFile, ok := app.fileExists("config.ini")

	if !ok {
		logger.Errorf("目录文件不存在")
		return false
	}

	C, err := goconfig.LoadConfigFile(configFile)
	if err != nil {
		logger.Errorf("load config file error %s", err)
		return false
	}

	app.id = C.MustValue("app", "id", "")
	app.key = C.MustValue("app", "key", "")
	app.secretKey = C.MustValue("app", "secretKey", "")
	return true
}
