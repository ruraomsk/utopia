package utopia

import (
	"fmt"
	"sync"
	"time"

	"github.com/ruraomsk/potop/hardware"
	"github.com/ruraomsk/potop/journal"
	"github.com/ruraomsk/potop/logger"
	"github.com/ruraomsk/potop/setup"
)

// From Spot to the controller					Reply from controller
// __________________________________________________________________________________
// TLC and group control: message 2				Status and detections: message 190
// (every second)
// ___________________________________________________________________________________
// Signal group count-down: message 8			Signal group feedback: message 4
// (every second)
// Extended count-down: message 9
// (every second)
// ___________________________________________________________________________________
// Diagnostic request: message 0				Extended diagnostic: message 5
// (event driven)
// ___________________________________________________________________________________
// Classified counts / vehicle length data		Classified counts / vehicle length data:
// request: message 24							message 24
// (periodic, default = 1 minute)
// ___________________________________________________________________________________
// Classified counts / vehicle speed data		Classified counts / vehicle length data:
// request: message 25							message 25
// (periodic, default = 1 minute)
// ___________________________________________________________________________________
// Special commands: message 6					Reply to a command: message 7
// ___________________________________________________________________________________
// Bus prediction: message 23					Bus detection: message 1
// (event driven)								or
// Time setting: message 3						Empty message (acknowledge)
// (periodic, default = 5 minutes)
// Empty message (polling)
// ___________________________________________________________________________________

var ctrl = ControllerUtopia{id: 1, status: 1, lastACK: 0, input: make([]byte, 0), output: make([]byte, 0)}
var mutex sync.Mutex
var delay chan Delay
var isUtopiaCtrl = false

type Delay struct {
	watchdog int
	ctrlSG   [64]bool
}
type messageUtopia struct {
	message []byte
}

func getDuration() time.Duration {
	return 25 * time.Second
}

var TestError = Testing{Work: false, State: 0, ErrorStatus: [3]byte{0, 0, 0}}
var live chan any
var executorLocal chan int

func GetControllerUtopia() ControllerUtopia {
	mutex.Lock()
	defer mutex.Unlock()
	return ctrl
}
func GetAutonom() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return ctrl.autonom
}
func SetAutonom(set bool) {
	mutex.Lock()
	defer mutex.Unlock()
	ctrl.autonom = set
}
func GetConnect() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return ctrl.connect
}
func controlUtopiaServer() {
	var timer *time.Timer
	logger.Info.Print("Нет управления от utopia")
	journal.SendMessage(journal.LevelUtopia, "Нет управления от utopia")
	for {
		<-live
		hardware.SetWork <- 1
		timer = time.NewTimer(getDuration())
		logger.Info.Print("Есть управление от utopia")
		journal.SendMessage(journal.LevelUtopia, "Есть управление от utopia")
		ctrl.connect = true
	loop:
		for {
			select {
			case <-timer.C:
				if !ctrl.autonom {
					hardware.SetWork <- 0
					hardware.CommandToKDM(0, 1)
					break loop
				}
			case <-live:
				timer.Stop()
				timer = time.NewTimer(getDuration())
				if !hardware.StateHardware.GetCentral() {
					hardware.SetWork <- 1
				}
			}
		}
		ctrl.connect = false
		logger.Error.Print("Потеряно управление от utopia")
		journal.SendMessage(journal.LevelUtopia, "Потеряно управление от utopia")
	}

}

func delays() {
	var newDelay Delay
	for {
		newDelay = <-delay
		if !hardware.IsConnectedKDM() {
			isUtopiaCtrl = false
			continue
		}
		if !isUtopiaCtrl && newDelay.ctrlSG != [64]bool{} {
			hardware.SetTLC(1, [64]bool{})
			time.Sleep(100 * time.Millisecond)
		}
		isUtopiaCtrl = true
		hardware.SetTLC(newDelay.watchdog, newDelay.ctrlSG)
		logger.Debug.Printf("Отправлено  %v", newDelay.ctrlSG[0:16])
	}
}

func Controller() {

	SetAutonom(setup.Set.Utopia.Avtonom)
	if setup.Set.Utopia.Avtonom {
		ctrl.status = 1
	}
	delay = make(chan Delay)
	live = make(chan any)
	executorLocal = make(chan int)

	journal.Setter <- journal.SetLevel{Level: journal.LevelUtopia, Double: true}

	go executor()
	go delays()
	go controlUtopiaServer()

	for {
		buffer := <-fromServer
		//Разберем
		messages := parserMessage(buffer)
		if hardware.IsConnectedKDM() {
			for _, v := range messages {
				workMessage(v)
			}
		} else {
			if setup.Set.Utopia.Replay {
				for _, v := range messages {
					workMessage(v)
				}
			}
		}
		toServer <- []byte{}
	}
}
func workMessage(message messageUtopia) {
	mutex.Lock()
	defer mutex.Unlock()
	ctrl.input = message.message
	logger.Debug.Printf("work [% 02X]", ctrl.input)
	if err := ctrl.verify(); err != nil {
		logger.Error.Printf("% 02X", ctrl.input)
		logger.Error.Print(err.Error())
		ctrl.sendNACK()
		return
	}
	if ctrl.isNak() {
		//Повторим предыдущее сообшение
		logger.Error.Printf("Повторяем сообщение % 02X", ctrl.output)
		toServer <- ctrl.output
		return
	}
	if ctrl.input[4] == ctrl.lastACK {
		logger.Error.Printf("ACK не изменился")
		ctrl.sendLive()
		return
	}
	if ctrl.isLive() {
		ctrl.sendLive()
		return
	}

	live <- 0
	if ctrl.input[5] == 10 {
		ctrl.input = fixSpotMessage(ctrl.input)
		ctrl.data = ctrl.input[6 : len(ctrl.input)-3]
	}
	switch ctrl.input[6] {
	case 2:
		// Message 2 – TLC and group control
		err := ctrl.TlcAndGroupControl.fromData(ctrl.data)
		if err != nil {
			logger.Error.Print(err.Error())
		}
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("<- %s", ctrl.TlcAndGroupControl.ToString()))
		ctrl.TlcAndGroupControl.execute()
		ctrl.StatusAndDetections.fill()
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("-> %s", ctrl.StatusAndDetections.ToString()))
		ctrl.sendReplay(ctrl.StatusAndDetections.toData())
	case 8:
		// Message 8 – Signal group count-down
		err := ctrl.CountDown.fromData(ctrl.data)
		if err != nil {
			logger.Error.Print(err.Error())
		}
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("<- %s", ctrl.CountDown.ToString()))
		ctrl.CountDown.execute()
		ctrl.SignalGroupFeedback.fill()
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("-> %s", ctrl.SignalGroupFeedback.ToString()))
		ctrl.sendReplay(ctrl.SignalGroupFeedback.toData())
	case 9:
		// Message 9 – Signal group extended count-down
		err := ctrl.ExtendedCountDown.fromData(ctrl.data)
		if err != nil {
			logger.Error.Print(err.Error())
		}
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("<- %s", ctrl.ExtendedCountDown.ToString()))
		ctrl.ExtendedCountDown.execute()
		ctrl.SignalGroupFeedback.fill()
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("-> %s", ctrl.SignalGroupFeedback.ToString()))
		ctrl.sendReplay(ctrl.SignalGroupFeedback.toData())
	case 3:
		// Message 3 – Date and time setting
		err := ctrl.DateAndTime.fromData(ctrl.data)
		if err != nil {
			logger.Error.Print(err.Error())
		}
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("<- %v", ctrl.DateAndTime.toData()))
		ctrl.sendLive()
	case 0:
		// Message 0 – Diagnostic request message
		err := ctrl.DiagnosticRequest.fromData(ctrl.data)
		if err != nil {
			logger.Error.Print(err.Error())
		}
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("<- %v", ctrl.DiagnosticRequest.toData()))
		ctrl.ExtendedDiagnostic.fill()
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("-> %v", ctrl.ExtendedDiagnostic.toData()))
		ctrl.sendReplay(ctrl.ExtendedDiagnostic.toData())
	case 24:
		// Message 24 – Request for classified counts by vehicle length
		err := ctrl.ReqClassifiedLenght.fromData(ctrl.data)
		if err != nil {
			logger.Error.Print(err.Error())
		}
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("<- %v", ctrl.ReqClassifiedLenght.toData()))
		ctrl.ClassifiedCounts.fill()
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("-> %v", ctrl.ClassifiedCounts.toData()))
		ctrl.sendReplay(ctrl.ClassifiedCounts.toData())
	case 25:
		// Message 25 – Request for classified counts by vehicle speed
		err := ctrl.ReqClassifiedSpeed.fromData(ctrl.data)
		if err != nil {
			logger.Error.Print(err.Error())
		}
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("<- %v", ctrl.ReqClassifiedSpeed.toData()))
		ctrl.ClassifiedSpeeds.fill()
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("-> %v", ctrl.ClassifiedSpeeds.toData()))
		ctrl.sendReplay(ctrl.ClassifiedSpeeds.toData())
	case 6:
		// Message 6 - Special commands
		err := ctrl.SpecialCommands.fromData(ctrl.data)
		if err != nil {
			logger.Error.Print(err.Error())
		}
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("<- %v", ctrl.SpecialCommands.toData()))
		ctrl.ReplaySpecial.fill()
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("-> %v", ctrl.ReplaySpecial.toData()))
		ctrl.sendReplay(ctrl.ReplaySpecial.toData())
	case 23:
		// Message 23 – Bus prediction
		err := ctrl.BusPrediction.fromData(ctrl.data)
		if err != nil {
			logger.Error.Print(err.Error())
		}
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("<- %v", ctrl.BusPrediction.toData()))
		ctrl.BusDetection.fill()
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("-> %v", ctrl.BusDetection.toData()))
		ctrl.sendReplay(ctrl.BusDetection.toData())
	default:
		logger.Error.Printf("Неопознанное сообщение от сервера %d", ctrl.input[6])
		journal.SendMessage(journal.LevelUtopia, fmt.Sprintf("<- Неопознанное сообщение от сервера %d", ctrl.input[6]))
		ctrl.sendLive()
	}

}
