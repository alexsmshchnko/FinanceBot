package logger

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"time"
)

type tLogLevel uint8

const (
	Error tLogLevel = iota
	Warning
	Debug
)

// const for log file path
const (
	winPath rune = 92 // \
	//fileName           = string(winPath) + "info"
	fileName      = "info"
	fileExtension = ".log"
)

func isValid(lvl tLogLevel) bool {
	switch lvl {
	case Error, Warning, Debug:
		return true
	default:
		return false
	}
}

type LogExt struct {
	*log.Logger
	logLevel tLogLevel
}

func newLog(file *os.File) (r *LogExt) {
	var logLvl tLogLevel

	switch os.Getenv("LOG_LEVEL") {
	case "DEBUG", "INFO":
		logLvl = 2
	case "WARNING", "WARN":
		logLvl = 1
	default:
		logLvl = 0
	}

	r = &LogExt{
		Logger:   log.New(file, "", log.LstdFlags|log.Lshortfile),
		logLevel: logLvl,
	}

	return
}

func (l *LogExt) setLogLevel(lvl tLogLevel) {
	if isValid(lvl) {
		l.logLevel = lvl
	}
}

func (l *LogExt) String() string {
	return fmt.Sprintf("Log Level = %d", l.logLevel)
}

var Log *LogExt

func init() {
	// set location of log file
	//var logpath = build.Default.GOPATH + "/src/chat/logger/info.log"
	var logpath = build.Default.Dir + fileName + time.Now().Format("02Jan150405") + fileExtension

	fmt.Println(logpath)

	flag.Parse()
	var file, err1 = os.Create(logpath)

	if err1 != nil {
		panic(err1)
	}

	Log = newLog(file)

	Log.Println("LogFile : " + logpath + "\n" + Log.String())
}
