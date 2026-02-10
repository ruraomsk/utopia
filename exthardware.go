package utopia

import (
	"time"

	"github.com/ruraomsk/potop/hardware"
	"github.com/ruraomsk/potop/logger"
)

func GetStatusDirs() []uint8 {
	// Оперативно считываем состояние направлений
	StateHardware := hardware.GetStateHard()
	if time.Since(StateHardware.LastOperation) > 100*time.Millisecond {
		hardware.CmdReadStatus <- 1
		<-hardware.ReplayReadStatus
		StateHardware = hardware.GetStateHard()
	}
	logger.Debug.Printf("Прочитано %v", StateHardware.StatusDirs)
	result := make([]uint8, 0)
	var b uint8
	for _, v := range StateHardware.StatusDirs {
		switch v {
		case 0:
			//   OFF = 0, //все сигналы выключены
			b = 0xE
		case 1:
			//   DEACTIV_YELLOW=1, //направление перешло в неактивное состояние, желтый после зеленого
			b = 0x1
		case 2:
			//   DEACTIV_RED=2, //направление перешло в неактивное состояние, красный
			b = 0x0
		case 3:
			//   ACTIV_RED=3, //направление перешло в активное состояние, красный
			b = 0x0
		case 4:
			//   ACTIV_REDYELLOW=4, //направление перешло в активное состояние, красный c желтым
			b = 0x2
		case 5:
			//   ACTIV_GREEN=5, //направление перешло в активное состояние, зеленый
			b = 0x8
		case 6:
			//   UNCHANGE_GREEN=6, //направление не меняло свое состояние, зеленый
			b = 0x8
		case 7:
			//   UNCHANGE_RED=7, //направление не меняло свое состояние, красный
			b = 0x0
		case 8:
			//   GREEN_BLINK=8, //зеленый мигающий сигнал
			b = 0xA
		case 9:
			//   ZM_YELLOW_BLINK=9, //желтый мигающий в режиме ЖМ
			b = 0x9
		case 10:
			//   OS_OFF=10,	//сигналы выключены в режиме ОС
			b = 0xe
		case 11:
			//   UNUSED=11 //неиспользуемое направление
			b = 0xf
		default:
			b = 0xe
		}
		result = append(result, b)
	}
	return result
}

func GetStatusUtopia() byte {
	StateHardware := hardware.GetStateHard()
	if !StateHardware.Connect {
		return 64
	}
	if StateHardware.Dark || StateHardware.Plan == 25 {
		return 6
	}
	if StateHardware.Flashing || StateHardware.Plan == 27 {
		return 3
	}
	if StateHardware.AllRed || StateHardware.Plan == 26 {
		return 4
	}
	if ctrl.status == 2 {
		return 2
	}
	if ctrl.status == 0 {
		return 1
	}
	if StateHardware.Plan != 0 {
		return 7
	}
	return byte(ctrl.status)
}
