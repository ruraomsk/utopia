package utopia

import (
	"fmt"
	"sync"
	"time"

	"github.com/ruraomsk/potop/hardware"
)

type Testing struct {
	mutex       sync.Mutex
	Work        bool
	State       byte
	ErrorStatus [3]byte
}

func (t *Testing) IsTest() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.Work
}
func (t *Testing) SetTest(set bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.Work = set
}

func (t *Testing) Lock() {
	t.mutex.Lock()
}
func (t *Testing) Unlock() {
	t.mutex.Unlock()
}

func toString(t time.Time) string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}
func bcc8(b byte, check uint16) uint16 {
	if b != 0 {
		var rbit uint16
		check = check ^ uint16(b)
		for i := 0; i < 8; i++ {
			rbit = check & 0x01
			check = check >> 1
			if rbit != 0 {
				check = check | 0x8000
				check = check ^ 0x2001
			}
		}
	}
	return check
}
func crc16_calc(data []byte) uint16 {
	var crc uint16
	crc = 0
	for _, v := range data {
		crc = bcc8(v, crc)
	}
	return crc
}
func getSingleError() byte {
	hs := hardware.GetStateHard()
	if hs.Key3 {
		return 64
	}

	switch hs.Status[0] {
	case 0:
		return 0
	case 1:
		return 1
	case 2:
		return 1
	case 3:
		return 16
	case 4:
		return 64
	case 5:
		return 1
	case 6:
		return 64
	case 7:
		return 64
	case 8:
		return 32
	case 9:
		return 32
	case 10:
		return 4
	case 11:
		return 4
	case 12:
		return 0
	case 13:
		return 32
	}
	return 0
}
func getExtendError() ExtError {
	hs := hardware.GetStateHard()
	if hs.Key3 {
		return ExtError{code: [3]byte{0, 1, 0}}
	}
	switch hs.Status[0] {
	case 0:
		return ExtError{code: [3]byte{0, 0, 0}}
	case 1:
		return ExtError{code: [3]byte{7, byte(hs.Status[1]), byte(hs.Status[2])}}
	case 2:
		return ExtError{code: [3]byte{9, byte(hs.Status[1]), byte(hs.Status[2])}}
	case 3:
		return ExtError{code: [3]byte{8, 1, 0}}
	case 4:
		return ExtError{code: [3]byte{12, 0, 0}}
	case 5:
		return ExtError{code: [3]byte{14, 1, 1}}
	case 6:
		return ExtError{code: [3]byte{0, 0, 0}}
	case 7:
		return ExtError{code: [3]byte{0, 0, 0}}
	case 8:
		return ExtError{code: [3]byte{1, 3, 1}}
	case 9:
		return ExtError{code: [3]byte{1, 3, 1}}
	case 10:
		return ExtError{code: [3]byte{2, 1, 0}}
	case 11:
		return ExtError{code: [3]byte{2, 1, 0}}
	case 12:
		return ExtError{code: [3]byte{0, 0, 0}}
	case 13:
		return ExtError{code: [3]byte{1, 3, 1}}
	}
	return ExtError{code: [3]byte{0, 0, 0}}
}
func parserMessage(buffer []byte) []messageUtopia {
	result := make([]messageUtopia, 0)
	for len(buffer) > 0 {
		count := 0
		for _, v := range buffer {
			if v != 0xff {
				break
			}
			count++
		}
		buffer = buffer[count:]
		l := buffer[5] + 9
		result = append(result, messageUtopia{message: buffer[0:l]})
		buffer = buffer[l:]
	}
	return result
}
func fixSpotMessage(data []byte) []byte {
	if len(data) != 19 || data[5] != 0x0A {
		return data
	}

	corrected := make([]byte, 20)

	// Копируем заголовок
	copy(corrected[:6], data[:6])

	// Увеличиваем длину на 1 (было 10, стало 11 байт данных)
	corrected[5] = 0x0B

	// Вставляем правильный Message ID (0x02 вместо 0x01)
	corrected[6] = 0x02

	// Signal groups (сдвигаем остальное)
	copy(corrected[7:], data[6:]) // 8 байт групп

	return corrected
}
