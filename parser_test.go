package utopia

import (
	"fmt"
	"strings"
	"testing"
)

var str = "FF FF FF FF FF 01 01 FE 00 06 0B 02 01 3C 33 00 00 00 00 00 00 00 03 CA A4 FF FF FF FF FF 01 01 FE 00 08 0A 01 3C 33 00 00 00 00 00 00 00 03 CC E2 FF FF FF FF FF 01 01 FE 00 06 0B 02 01 3C 33 00 00 00 00 00 00 00 03 CA A4 FF FF FF FF FF 01 01 FE 00 08 0A 01 3C 33 00 00 00 00 00 00 00 03 CC E2 FF FF FF FF FF 01 01 FE 00 06 0B 02 01 3C 33 00 00 00 00 00 00 00 03 CA A4 FF FF FF FF FF 01 01 FE 00 08 0A 01 24 00 00 00 00 00 00 00 00 03 E9 23 FF FF FF FF FF 01 01 FE 00 06 0B 02 01 24 00 00 00 00 00 00 00 00 03 6F 2A FF FF FF FF FF 01 01 FE 00 08 0A 01 24 4C 00 00 00 00 00 00 00 03 2C A9 FF FF FF FF FF 01 01 FE 00 06 0B 02 01 24 4C 00 00 00 00 00 00 00 03 2A EF FF FF FF FF FF 01 01 FE 00 08 0A 01 24 4C 00 00 00 00 00 00 00 03 2C A9 FF FF FF FF FF 01 01 FE 00 06 0B 02 01 24 4C 00 00 00 00 00 00 00 03 2A EF FF FF FF FF FF 01 01 FE 00 08 0A 01 24 4C 00 00 00 00 00 00 00 03 2C A9 FF FF FF FF FF 01 01 FE 00 06 0B 02 01 24 4C 00 00 00 00 00 00 00 03 2A EF FF FF FF FF FF 01 01 FE 00 08 0A 01 24 4C 00 00 00 00 00 00 00 03 2C A9 FF FF FF FF FF 01 01 FE 00 06 0B 02 01 24 4C 00 00 00 00 00 00 00 03 2A EF FF FF FF FF FF 01 01 FE 00 08 0A 01 3C 00 00 00 00 00 00 00 00 03 E3 23 FF FF FF FF FF 01 01 FE 00 06 0B 02 01 3C 00 00 00 00 00 00 00 00 03 65 2A FF FF FF FF FF 01 01 FE 00 08 0A 01 3C 33 00 00 00 00 00 00 00 03 CC E2"

func TestParserBuffer(t *testing.T) {
	buffer, _ := parseHexString(str)

	messages := parserMessage(buffer)
	for _, v := range messages {
		t.Errorf("% 02X", v)
		if v.message[5] == 0x0a {
			res := fixSpotMessage(v.message)
			t.Errorf("% 02X!", res)
		}
		ctrl.input = v.message
		err := ctrl.verify()
		if err != nil {
			t.Errorf("% 02X", v)
			t.Errorf(err.Error())
		}
	}
}
func parseHexString(hexStr string) ([]byte, error) {
	// Удаляем все пробелы и разбиваем по пробелам
	parts := strings.Fields(hexStr)

	// Создаем слайс байт нужного размера
	result := make([]byte, len(parts))

	// Преобразуем каждую часть
	for i, part := range parts {
		// Проверяем, что часть имеет длину 2 символа (FF, 01 и т.д.)
		if len(part) != 2 {
			return nil, fmt.Errorf("invalid hex part at position %d: %s", i, part)
		}

		// Преобразуем строку в байт
		var b byte
		_, err := fmt.Sscanf(part, "%02X", &b)
		if err != nil {
			return nil, fmt.Errorf("failed to parse hex at position %d: %v", i, err)
		}
		result[i] = b
	}

	return result, nil
}
