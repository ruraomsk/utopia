package utopia

import (
	"github.com/ruraomsk/potop/hardware"
	"github.com/ruraomsk/potop/logger"
)

func executor() {
	for {
		<-executorLocal
		if isLocalWork {
			continue
		}
		logger.Debug.Printf("Выполняем команду Переход в локальный режим")
		hardware.CommandToKDM(1, 0)
		isUtopiaCtrl = false
		isLocalWork = true
	}
}
