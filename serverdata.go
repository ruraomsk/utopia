package utopia

import (
	"fmt"
)

type ServerUtopia struct {
	id      byte   //Идентификатор контроллера
	lastACK byte   //Предыдущий АСК
	input   []byte //Принято от контроллера
	output  []byte //Передано на контроллер
	data    []byte //Вычитанные чистые данные
	// lastTime time.Time //Последний прием от контроллера
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

func (c *ServerUtopia) sendNACK() {
	c.sendMessage(21, make([]byte, 0))
}
func (c *ServerUtopia) getACK() byte {
	var cnt byte
	if c.lastACK == 6 {
		cnt = 8
	} else {
		cnt = 6
	}
	c.lastACK = byte(cnt)
	return cnt
}

func (c *ServerUtopia) sendLive() {
	cnt := c.getACK()
	c.sendMessage(byte(cnt), make([]byte, 0))
}
func (c *ServerUtopia) sendCommand(message []byte) {
	// fmt.Printf("Command % 02X \n", message)
	c.sendMessage(0, message)
}
func (c *ServerUtopia) sendMessage(cnt byte, message []byte) {
	if cnt != 21 && cnt != 6 && cnt != 8 {
		cnt = c.getACK()
	}
	c.output = make([]byte, 0)
	c.output = append(c.output, 1, 1, 0xfe, 0, cnt, byte(len(message)))
	c.output = append(c.output, message...)
	crc := crc16_calc(c.output[4:])
	c.output = append(c.output, 3, byte((crc>>8)&0xff), byte(crc&0xff))
	toController <- c.output
}
func (c *ServerUtopia) verify() error {
	if c.input[0] != 1 || (c.input[1]+c.input[2]) != 0xff {
		return fmt.Errorf("неверный признак сообшения от СПОТ")
	}
	if c.input[3] != c.id {
		return fmt.Errorf("неверный номер контроллера")
	}
	if c.input[4] != 6 && c.input[4] != 8 && c.input[4] != 21 {
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
func (c *ServerUtopia) isLive() bool {
	return c.input[5] == 0
}
func (c *ServerUtopia) isNak() bool {
	return c.input[4] == 21
}
