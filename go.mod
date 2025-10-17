module github.com/ruraomsk/utopia

go 1.23.3

require (
	github.com/goburrow/serial v0.1.0
	github.com/ruraomsk/ag-server v0.0.0
	github.com/ruraomsk/potop v0.0.0
)

replace github.com/ruraomsk/ag-server v0.0.0 => ../ag-server

replace github.com/ruraomsk/potop v0.0.0 => ../potop
