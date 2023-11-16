package gwlog

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"gitee.com/arya123/chirpstack-api/go/gw"
	"github.com/brocaar/lorawan"
)

// --- Types -------------------------------
type Module = string
type Level = int32
type GwLog = gw.GatewayLogger

const (
	PANIC = iota
	FATAL
	ERROR
	WARN
	INFO
	DEBUG
)

type GwLogger struct {
	exitChan chan bool
	logChan  chan GwLog
	logConn  *net.UDPConn
	module   string

	wg sync.WaitGroup
}

type GwLoggerConfig struct {
	RemoteIP   string
	RemotePort int
	Module     string
}

// --- Const -------------------------------

const (
	Deveui = "deveui: "
	Gwid   = "gatewayid: "
)

// --- Variables ---------------------------

var (
	gwlogger *GwLogger = nil
)

// --- Functions ---------------------------

func NewGwLogger(config *GwLoggerConfig) *GwLogger {
	addr := &net.UDPAddr{IP: net.ParseIP(config.RemoteIP), Port: config.RemotePort}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic(err)
	}

	logger := GwLogger{
		exitChan: make(chan bool),
		logChan:  make(chan GwLog, 10),
		logConn:  conn,
		module:   config.Module,
	}

	logger.Start()

	return &logger
}

func SetGwLogger(logger *GwLogger) {
	gwlogger = logger
}

func GetGwLogger() *GwLogger {
	if gwlogger == nil {
		panic("GwLogger is not initialized")
	}
	return gwlogger
}

func Label(arg ...string) []string {
	return arg
}

func FmtEUI(euitype string, eui lorawan.EUI64) string {
	return fmt.Sprintf(euitype, eui.String())
}

func (logger *GwLogger) Log(level Level, label []string, msg string, args ...interface{}) {
	log := GwLog{
		Timestamp: time.Now().Unix(),
		Level:     level,
		Module:    logger.module,
		Label:     label,
		Message:   fmt.Sprintf(msg, args...),
	}
	logger.logChan <- log
}

func (logger *GwLogger) Start() {
	logger.wg.Add(1)
	go func() {
		logger.logForwarder()
		logger.wg.Done()
	}()
}

func (logger *GwLogger) Stop() {
	logger.exitChan <- true
	logger.wg.Wait()
}

func (logger *GwLogger) logForwarder() {
	for {
		select {
		case log := <-gwlogger.logChan:
			j, err := json.Marshal(log)
			if err != nil {
				continue
			}
			logger.logConn.Write(j)

		case <-gwlogger.exitChan:
			goto exit
		}
	}

exit:
	logger.logConn.Close()
	close(gwlogger.logChan)
	close(gwlogger.exitChan)
}
