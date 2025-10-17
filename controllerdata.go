package utopia

import (
	"fmt"
)

type ControllerUtopia struct {
	id      byte   //Идентификатор контроллера
	lastACK byte   //Предыдущий АСК
	input   []byte //Принято от сервера
	output  []byte //Передано на сервер
	data    []byte //Вычитанные чистые данные
	status  int    //0 1 - Local 2  -central
	autonom bool
	connect bool
	//Запросы сервера
	TlcAndGroupControl  TlcAndGroupControl  // Spot TLC and group control (2)
	CountDown           CountDown           // Spot Message 8  – Signal group count-down, управление сигнальными группами
	ExtendedCountDown   ExtendedCountDown   // Spot Message 9  – Extended Signal group count-down, управление сигнальными группами
	DiagnosticRequest   DiagnosticRequest   // Message 0 – Diagnostic request message
	ReqClassifiedLenght ReqClassifiedLength // Message 24 – Request for classified counts by vehicle length
	ReqClassifiedSpeed  ReqClassifiedSpeed  // Message 25 – Request for classified counts by vehicle speed
	BusPrediction       BusPrediction       // Message 23 – Bus prediction
	DateAndTime         DateAndTime         // Message 3 – Date and time setting
	SpecialCommands     SpecialCommands     // Message 6 - Special commands
	//Ответы контроллера
	StatusAndDetections StatusAndDetections // Controller Message 190 - Status and detections: message
	SignalGroupFeedback SignalGroupFeedback // Controller Message 4 – Signal Group feedback, ответ от сигнальных групп
	ExtendedDiagnostic  ExtendedDiagnostic  // Message 5 – Extended diagnostic
	ClassifiedCounts    ClassifiedCounts    // Message 24 – Classified counts by vehicle length
	ClassifiedSpeeds    ClassifiedSpeeds    // Message 25 – Classified counts by vehicle speed
	BusDetection        BusDetection        // Message 1 – Bus detection
	ReplaySpecial       ReplaySpecial       // Message 7 – Reply to a special command
}

func (c *ControllerUtopia) sendNACK() {
	c.sendMessage(21, make([]byte, 0))
}
func (c *ControllerUtopia) sendLive() {
	cnt := 6
	if c.input[5] == 6 {
		cnt = 8
	}
	c.sendMessage(byte(cnt), make([]byte, 0))
}
func (c *ControllerUtopia) sendReplay(message []byte) {
	// fmt.Printf("Replay % 02X \n", message)
	c.sendMessage(0, message)
}
func (c *ControllerUtopia) sendMessage(cnt byte, message []byte) {
	if cnt != 21 && cnt != 6 && cnt != 8 {
		cnt = 6
		if c.input[5] == 6 {
			cnt = 8
		}
	}
	c.output = make([]byte, 0)
	c.output = append(c.output, 1, 0, 0xff, c.id, cnt, byte(len(message)))
	c.output = append(c.output, message...)
	crc := crc16_calc(c.output[4:])
	c.output = append(c.output, 3, byte((crc>>8)&0xff), byte(crc&0xff))
	toServer <- c.output
}
func (c *ControllerUtopia) verify() error {
	count := 0
	for _, v := range c.input {
		if v != 0xff {
			break
		}
		count++
	}
	c.input = c.input[count:]
	// logger.Info.Printf("% 02X", c.input)

	if len(c.input) < 9 {
		return fmt.Errorf("неверная длина сообшения от спот %d", len(c.input))
	}
	if c.input[0] != 1 || (c.input[1]+c.input[2]) != 0xff {
		return fmt.Errorf("неверный признак сообшения от спот")
	}
	if c.input[3] != 0 {
		return fmt.Errorf("неверный ORG СПОТ")
	}
	if c.input[4] != 6 && c.input[4] != 8 {
		return fmt.Errorf("неверный тип сообщения")
	}
	l := int(c.input[5]) + 9
	if l != len(c.input) {
		return fmt.Errorf("неверная длина сообщения %d должна быть %d", c.input[5], len(c.input)-9)
	}
	if c.input[l-3] != 3 {
		return fmt.Errorf("неверный код EXT ")
	}
	crc := crc16_calc(c.input[4 : len(c.input)-3])
	icrc := uint16(c.input[l-2])<<8 | uint16(c.input[l-1])
	if crc != icrc {
		return fmt.Errorf("неверная CRC %X должна быть %X", icrc, crc)
	}
	c.data = c.input[6 : len(c.input)-3]
	return nil
}
func (c *ControllerUtopia) isLive() bool {
	return c.input[5] == 0
}
func (c *ControllerUtopia) isNak() bool {
	return c.input[4] == 21
}
