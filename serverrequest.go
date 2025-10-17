package utopia

import (
	"fmt"
	"time"

	"github.com/ruraomsk/potop/hardware"
	"github.com/ruraomsk/potop/setup"
)

// TlcAndGroupControl Spot TlcAndGroupControl and group control (2)
type TlcAndGroupControl struct {
	lastop   time.Time
	command  int      //команда 0 нет команды 1 - локальная команда 2 - команда из центра 3 - мигание 6 - отключение 7-запустить план
	watchdog int      //Watch-dog время действия (в секундах) для команды
	ctrlSG   [64]bool //управление сигнальными группами(СГ).
	// Информация кодируется с использованием одного бита для передачи команды управления одной группой сигналов.
	// Бит устанавливается в 1, если группа управляется ЗЕЛЕНЫМ, устанавливается в 0 если группа получает команду КРАСНЫЙ.
}

func (t *TlcAndGroupControl) ToString() string {
	res := fmt.Sprintf("Message 02 %s %d %d [ ", toString(t.lastop), t.command, t.watchdog)
	for _, v := range t.ctrlSG {
		if !v {
			res += "X"
		} else {
			res += "_"
		}
	}
	res += " ]"
	return res
}
func (t *TlcAndGroupControl) execute() {
	// logger.Debug.Printf("execute TlcAndGroupControl %v", t)
	if ctrl.autonom {
		useDelay.clear()
		return
	}

	ctrl.status = t.command
	if t.command == 0 {
		useDelay.clear()
		return
	}
	if t.command == 1 {
		hardware.CommandToKDM(1, 0)
		useDelay.clear()
		return
	}
	if t.command == 2 {
		delay <- Delay{watchdog: t.watchdog, ctrlSG: t.ctrlSG}
	}
	if t.command == 3 {
		hardware.CommandToKDM(1, 0)
		hardware.CommandToKDM(3, 1)
		useDelay.clear()
	}
	if t.command == 6 {
		hardware.CommandToKDM(1, 0)
		hardware.CommandToKDM(3, 0)
		useDelay.clear()
	}
	if t.command == 7 {
		hardware.CommandToKDM(1, 0)
		hardware.CommandToKDM(7, t.watchdog)
		useDelay.clear()
	}

}
func (t *TlcAndGroupControl) toData() []byte {
	t.lastop = time.Now()
	result := make([]byte, 0)
	result = append(result, 2, byte(t.command), byte(t.watchdog))
	b := make([]byte, 8)
	if setup.Set.Utopia.Recode {
		//Через задницу
		j := 0 //Позиция в результате
		l := 0 //Номер бита
		for i := 0; i < len(t.ctrlSG); i++ {
			d := 0
			if t.ctrlSG[i] {
				d = 1
			}
			d = d << l
			b[j] |= byte(d)
			l++
			if l > 7 {
				j++
				l = 0
			}
		}
		result = append(result, b...)
		return result
	} else {
		//По феншую
		j := 0 //Позиция в результате
		l := 7 //Номер бита
		for i := 0; i < len(t.ctrlSG); i++ {
			d := 0
			if t.ctrlSG[i] {
				d = 1
			}
			d = d << l
			b[j] |= byte(d)
			l--
			if l < 0 {
				j++
				l = 7
			}
		}
		result = append(result, b...)
		return result
	}
}
func (t *TlcAndGroupControl) fromData(data []byte) error {
	if data[0] != 2 {
		return fmt.Errorf("это не сообщение 2")
	}
	if len(data) != 11 {
		return fmt.Errorf("сообщение 2 неверная длина")
	}
	t.lastop = time.Now()
	t.command = int(data[1])
	t.watchdog = int(data[2])
	if setup.Set.Utopia.Recode {
		//Через задницу
		j := 0
		l := 0
		for i := 0; i < len(t.ctrlSG); i++ {
			if (data[3+j]>>l)&1 > 0 {
				t.ctrlSG[i] = true
			} else {
				t.ctrlSG[i] = false
			}
			l++
			if l > 7 {
				j++
				l = 0
			}
		}
		return nil
	} else {
		//По феншую
		j := 0
		l := 7
		for i := 0; i < len(t.ctrlSG); i++ {
			if (data[3+j]>>l)&1 > 0 {
				t.ctrlSG[i] = true
			} else {
				t.ctrlSG[i] = false
			}
			l--
			if l < 0 {
				j++
				l = 7
			}
		}
		return nil
	}
}

// CountDown Spot Message 8  – Signal group count-down, управление сигнальными группами
type CountDown struct {
	lastop time.Time
	index  int      //Индекс группы из 8 сигнальных групп
	counts [64]byte //Обратный отсчет для группы сигналов. Один байт для каждой СГ.
	//Время работы (в секундах, до следующего изменения командой).
	// Ожидаемое время устанавливается равным 255, если сигнальная группа не доступна.
}

func (c CountDown) ToString() string {
	return fmt.Sprintf("Message 08 %s %v ", toString(c.lastop), c.counts)
}

func (c CountDown) execute() {
	// logger.Debug.Printf("execute CountDown %v", c)
	if ctrl.status == 2 {
		hardware.SetSignalCountDown(c.counts)
	}
}

func (c CountDown) toData() []byte {
	// c.lastop = time.Now()
	result := make([]byte, 0)
	result = append(result, 8, byte(c.index))
	// if setup.Set.Utopia.Recode {
	// 	for i := 7; i < len(c.counts); i += 8 {
	// 		for j := i; j > i-8; j-- {
	// 			result = append(result, c.counts[j])
	// 		}
	// 	}
	// } else {
	for _, v := range c.counts {
		result = append(result, v)
	}
	// }
	return result
}
func (c *CountDown) fromData(data []byte) error {
	if data[0] != 8 {
		return fmt.Errorf("это не сообщение 8")
	}
	c.lastop = time.Now()
	c.index = int(data[1])
	// if setup.Set.Utopia.Recode {
	// 	k := 0
	// 	for i := 7; i < len(c.counts); i += 8 {
	// 		for j := i; j > i-8; j-- {
	// 			c.counts[k] = data[j+2]
	// 			logger.Debug.Printf("i=%d j=%d k=%d %d", i, j, k, c.counts[k])
	// 			k++
	// 		}
	// 	}
	// 	logger.Debug.Printf("counts %v", c.counts)
	// } else {
	for i := 0; i < len(c.counts); i++ {
		c.counts[i] = data[i+2]
	}
	// }
	return nil
}

// ExtendedCountDown Spot Message 9  – Extended Signal group count-down, управление сигнальными группами
type ExtendedCountDown struct {
	lastop     time.Time
	plan       int
	index      int //Индекс группы из 8 сигнальных групп
	stage      int //current stage length, according to signal plan
	signalplan int //Signal plan command source, that has
	// requested the activation of the current
	// signal plan:
	// 1 = Calendar
	// 2 = Traffic Scenario
	// 3 = Operator
	spare  [5]byte
	counts [64]byte //Обратный отсчет для группы сигналов. Один байт для каждой СГ.
	//Время работы (в секундах, до следующего изменения командой).
	// Ожидаемое время устанавливается равным 255, если сигнальная группа не доступна.
}

func (e *ExtendedCountDown) ToString() string {
	return fmt.Sprintf("Message 09 %s %v ", toString(e.lastop), e.counts)
}

func (e *ExtendedCountDown) execute() {
	// logger.Debug.Printf("execute ExtendedCountDown")
	if ctrl.status == 2 {
		hardware.SetSignalCountDown(e.counts)
	}
}

func (e *ExtendedCountDown) toData() []byte {
	e.lastop = time.Now()
	result := make([]byte, 0)
	result = append(result, 9, byte(e.plan), byte(e.index), byte(e.stage), byte(e.signalplan))
	for _, v := range e.spare {
		result = append(result, v)
	}
	// if setup.Set.Utopia.Recode {
	// 	for i := 7; i < len(e.counts); i += 8 {
	// 		for j := i; j > i-8; j-- {
	// 			result = append(result, e.counts[j])
	// 		}
	// 	}
	// } else {
	for _, v := range e.counts {
		result = append(result, v)
	}
	// }
	return result
}
func (e *ExtendedCountDown) fromData(data []byte) error {
	// logger.Debug.Printf("fromData ExtendedCountDown start")
	if data[0] != 9 {
		return fmt.Errorf("это не сообщение 9")
	}
	e.lastop = time.Now()
	e.plan = int(data[1])
	e.index = int(data[2])
	e.stage = int(data[3])
	e.signalplan = int(data[4])
	for i := 0; i < len(e.spare); i++ {
		e.spare[i] = data[i+5]
	}
	// if setup.Set.Utopia.Recode {
	// 	k := 0
	// 	for i := 7; i < len(e.counts); i += 8 {
	// 		for j := i; j > i-8; j-- {
	// 			e.counts[k] = data[j+10]
	// 			k++
	// 		}
	// 	}
	// } else {
	for i := 0; i < len(e.counts); i++ {
		e.counts[i] = data[i+10]
	}
	// }
	// logger.Debug.Printf("fromData ExtendedCountDown stop")
	return nil
}

// DiagnosticRequest Message 0 – Diagnostic request message
type DiagnosticRequest struct {
	lastop time.Time
}

func (d *DiagnosticRequest) toData() []byte {
	d.lastop = time.Now()
	result := make([]byte, 0)
	result = append(result, 0)
	return result
}
func (d *DiagnosticRequest) fromData(data []byte) error {
	if data[0] != 0 {
		return fmt.Errorf("это не сообщение 0")
	}
	d.lastop = time.Now()
	return nil
}

// ReqClassifiedLength Message 24 – Request for classified counts by vehicle length
type ReqClassifiedLength struct {
	lastop time.Time
}

func (r *ReqClassifiedLength) toData() []byte {
	r.lastop = time.Now()
	result := make([]byte, 0)
	result = append(result, 24)
	return result
}
func (r *ReqClassifiedLength) fromData(data []byte) error {
	if data[0] != 24 {
		return fmt.Errorf("это не сообщение 24")
	}
	r.lastop = time.Now()
	return nil
}

// ReqClassifiedSpeed Message 25 – Request for classified counts by vehicle speed
type ReqClassifiedSpeed struct {
	lastop time.Time
}

func (r *ReqClassifiedSpeed) toData() []byte {
	r.lastop = time.Now()
	result := make([]byte, 0)
	result = append(result, 25)
	return result
}
func (r *ReqClassifiedSpeed) fromData(data []byte) error {
	if data[0] != 25 {
		return fmt.Errorf("это не сообщение 25")
	}
	r.lastop = time.Now()
	return nil
}

// BusPrediction Message 23 – Bus prediction
type BusPrediction struct {
	lastop         time.Time
	PTcode         int  //PT service code
	PTid           int  //PT vehicle ID
	PTtime         int  //PT vehicle expected arrival time
	Expected       int  //Expected arrival time standard deviation (seconds)
	Estimated      int  //Estimated waiting time at the stop 	line
	Requestedgroup byte //Requested group (0 if unknown)
}

func (b *BusPrediction) toData() []byte {
	b.lastop = time.Now()
	result := make([]byte, 0)
	result = append(result, 23)
	result = append(result, byte((b.PTcode>>8)&0xff), byte(b.PTcode&0xff))
	result = append(result, byte((b.PTid>>8)&0xff), byte(b.PTid&0xff))
	result = append(result, byte((b.PTtime>>8)&0xff), byte(b.PTtime&0xff))
	result = append(result, byte((b.Expected>>8)&0xff), byte(b.Expected&0xff))
	result = append(result, byte((b.Estimated>>8)&0xff), byte(b.Estimated&0xff))
	result = append(result, b.Requestedgroup)
	return result
}
func (b *BusPrediction) fromData(data []byte) error {
	if data[0] != 23 {
		return fmt.Errorf("это не сообщение 23")
	}
	b.lastop = time.Now()
	b.PTcode = int(data[1])<<8 | int(data[2])
	b.PTid = int(data[3])<<8 | int(data[4])
	b.PTtime = int(data[5])<<8 | int(data[6])
	b.Expected = int(data[7])<<8 | int(data[8])
	b.Estimated = int(data[9])<<8 | int(data[10])
	b.Requestedgroup = data[11]
	return nil
}

// DateAndTime Message 3 – Date and time setting
type DateAndTime struct {
	DateTime time.Time
}

func (d *DateAndTime) toData() []byte {
	result := make([]byte, 0)
	result = append(result, 3)
	result = append(result, byte(d.DateTime.Year()%100))
	result = append(result, byte(d.DateTime.Month()))
	result = append(result, byte(d.DateTime.Day()))
	result = append(result, byte(d.DateTime.Weekday()))
	result = append(result, byte(d.DateTime.Hour()))
	result = append(result, byte(d.DateTime.Minute()))
	result = append(result, byte(d.DateTime.Second()))
	return result
}
func (d *DateAndTime) fromData(data []byte) error {
	if data[0] != 3 {
		return fmt.Errorf("это не сообщение 3")
	}
	year := int(data[1]) + 2000
	month := int(data[2])
	day := int(data[3])
	hour := int(data[5])
	minute := int(data[6])
	seconds := int(data[7])
	d.DateTime = time.Date(year, time.Month(month), day, hour, minute, seconds, 0, time.Local)
	return nil
}

// SpecialCommands Message 6 - Special commands
type SpecialCommands struct {
	lastop time.Time
	value  byte //value 1 for reset alarms
}

func (s *SpecialCommands) toData() []byte {
	s.lastop = time.Now()
	result := make([]byte, 0)
	result = append(result, 6, s.value)

	return result
}
func (s *SpecialCommands) fromData(data []byte) error {
	if data[0] != 6 {
		return fmt.Errorf("это не сообщение 6")
	}
	s.lastop = time.Now()
	s.value = data[1]
	return nil
}
