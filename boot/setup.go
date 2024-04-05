package boot

import "whatsapp_multi_session_general/config"

func Setup() {
	//initialize config
	config.Initialize()

	//// listener on trigger start up
	//listener.TriggerStartUp()
	////listener on trigger shutdown
	//listener.ListenForShutdownEvent()
}
