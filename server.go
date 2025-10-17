package utopia

import (
	"math/rand"
	"time"

	"github.com/ruraomsk/ag-server/logger"
)

// From Spot to the controller					Reply from controller
// __________________________________________________________________________________
// TLC and group control: message 2				Status and detections: message 190
// (every second)
//___________________________________________________________________________________
// Signal group count-down: message 8			Signal group feedback: message 4
// (every second)
// Extended count-down: message 9
// (every second)
//___________________________________________________________________________________
// Diagnostic request: message 0				Extended diagnostic: message 5
// (event driven)
//___________________________________________________________________________________
// Classified counts / vehicle length data		Classified counts / vehicle length data:
// request: message 24							message 24
// (periodic, default = 1 minute)
//___________________________________________________________________________________
// Classified counts / vehicle speed data		Classified counts / vehicle length data:
// request: message 25							message 25
// (periodic, default = 1 minute)
//___________________________________________________________________________________
// Special commands: message 6					Reply to a command: message 7
//___________________________________________________________________________________
// Bus prediction: message 23					Bus detection: message 1
// (event driven)								or
// Time setting: message 3						Empty message (acknowledge)
// (periodic, default = 5 minutes)
// Empty message (polling)
//___________________________________________________________________________________

var serv = ServerUtopia{id: 1, lastACK: 0}

func listenController() {
	for {
		serv.input = <-fromController
		// fmt.Printf("-> % 02X \n", serv.input)
		if err := serv.verify(); err != nil {
			serv.sendNACK()
			logger.Error.Print(err.Error())
			continue
		}
		if serv.isNak() {
			//Повторим предыдущее сообшение
			logger.Error.Printf("Повторяем сообщение % 02X", serv.output)
			toController <- serv.output
			continue
		}
		if serv.isLive() {
			continue
		}
		// fmt.Printf("serv  % 02X \n", serv.data)
		switch serv.input[6] {
		case 190:
			err := serv.StatusAndDetections.fromData(serv.data)
			if err != nil {
				logger.Error.Print(err.Error())
			}
		case 4:
			err := serv.SignalGroupFeedback.fromData(serv.data)
			if err != nil {
				logger.Error.Print(err.Error())
			}
		case 5:
			err := serv.ExtendedDiagnostic.fromData(serv.data)
			if err != nil {
				logger.Error.Print(err.Error())
			}
		case 24:
			err := serv.ClassifiedCounts.fromData(serv.data)
			if err != nil {
				logger.Error.Print(err.Error())
			}
		case 25:
			err := serv.ClassifiedSpeeds.fromData(serv.data)
			if err != nil {
				logger.Error.Print(err.Error())
			}
		case 7:
			err := serv.ReplaySpecial.fromData(serv.data)
			if err != nil {
				logger.Error.Print(err.Error())
			}
		case 1:
			err := serv.BusDetection.fromData(serv.data)
			if err != nil {
				logger.Error.Print(err.Error())
			}

		default:
			logger.Error.Printf("Неопознанный ответ от контроллера %d", serv.input[6])
		}

	}
}
func Server() {
	go listenController()
	ticker := time.NewTicker(time.Second)
	sendTLC := time.NewTimer(time.Second)
	sendCountDown := time.NewTicker(5 * time.Second)
	sendExtebdedCountDown := time.NewTicker(6 * time.Second)
	sendBus := time.NewTicker(20 * time.Second)
	sendTime := time.NewTicker(1 * time.Minute)
	sendCount := time.NewTicker(21 * time.Second)
	sendSpeed := time.NewTicker(22 * time.Second)
	sendSpecial := time.NewTicker(23 * time.Second)
	sendDiagnostic := time.NewTicker(24 * time.Second)
	for {
		select {
		case <-ticker.C:
			serv.sendLive()
		case <-sendTime.C:
			serv.DateAndTime.DateTime = time.Now()
			serv.sendCommand(serv.DateAndTime.toData())
		case <-sendTLC.C:
			serv.TlcAndGroupControl.command = rand.Intn(7)
			serv.TlcAndGroupControl.watchdog = rand.Intn(60)
			for i := 0; i < len(serv.TlcAndGroupControl.ctrlSG); i++ {
				if rand.Intn(10) < 5 {
					serv.TlcAndGroupControl.ctrlSG[i] = false
				} else {
					serv.TlcAndGroupControl.ctrlSG[i] = true
				}
			}
			serv.sendCommand(serv.TlcAndGroupControl.toData())
			sendTLC = time.NewTimer(time.Duration(serv.TlcAndGroupControl.watchdog) * time.Second)
		case <-sendCountDown.C:
			for i := 0; i < len(serv.CountDown.counts); i++ {
				serv.CountDown.counts[i] = byte(rand.Intn(255))
			}
			serv.sendCommand(serv.CountDown.toData())
		case <-sendExtebdedCountDown.C:
			serv.sendCommand(serv.ExtendedCountDown.toData())
		case <-sendBus.C:
			serv.sendCommand(serv.BusPrediction.toData())
		case <-sendCount.C:
			serv.sendCommand(serv.ReqClassifiedLenght.toData())
		case <-sendSpeed.C:
			serv.sendCommand(serv.ReqClassifiedSpeed.toData())
		case <-sendSpecial.C:
			serv.sendCommand(serv.SpecialCommands.toData())
		case <-sendDiagnostic.C:
			serv.sendCommand(serv.DiagnosticRequest.toData())
		}

	}
}
