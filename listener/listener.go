package listener

import (
	"fmt"

	"whatsapp_multi_session_general/commandhandler"
	"whatsapp_multi_session_general/config"
	"whatsapp_multi_session_general/primitive"

	"github.com/gookit/event"
)

type Listener struct {
	CommandHandler commandhandler.CommandHandler
}

func NewListener(commandhandler commandhandler.CommandHandler) Listener {
	return Listener{
		CommandHandler: commandhandler,
	}
}

// TriggerStartUp sends a signal to the repository and performs start up actions.
// this call should be not initiated on event because we can just call it on the main.go
func (l Listener) TriggerStartUp() {
	if config.Conf.StartUp.EnableAutoLogin {
		fmt.Println("trigger TriggerStartUp for EnableAutoLogin is enabled")
		l.CommandHandler.AutoLogin()
	}
}

// ListenForShutdownEvent listen on the shutdown event
// look utils/ShutDownEvent constant.
func (l Listener) ListenForShutdownEvent() {
	event.On(primitive.ShutDownEvent, event.ListenerFunc(func(e event.Event) error {
		// TriggerShutdown wrapping action for the shutdown event
		l.TriggerShutDown()
		return nil
	}))
}

// TriggerShutDown sends a signal to the code handler and performs shutdown actions.
// this call should be not initiated on event because we can just call it on the main.go
func (l Listener) TriggerShutDown() {
	//add feature flag and add handler for the code
	if config.Conf.ShutDown.EnableAutoShutDown && config.Conf.AutoDisconnect {
		fmt.Println("trigger TriggerStartUp for EnableAutoShutDown is enabled and config.Conf.AutoDisconnect")
		l.CommandHandler.AutoDisconnect()
	}

	if config.Conf.ShutDown.EnableAutoShutDown && !config.Conf.AutoDisconnect && config.Conf.AutoLogout {
		fmt.Println("trigger TriggerStartUp for EnableAutoShutDown is enabled and config.Conf.AutoLogout")
		l.CommandHandler.AutoLogOut()
	}
}
