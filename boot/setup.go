package boot

import (
	"fmt"

	"whatsapp_multi_session_general/commandhandler"
	"whatsapp_multi_session_general/config"
	"whatsapp_multi_session_general/database"
	"whatsapp_multi_session_general/handler"
	"whatsapp_multi_session_general/listener"
	"whatsapp_multi_session_general/routers"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine) *gin.Engine {
	//initialize config
	config.Initialize()

	//initiate database sqlite
	sqliteConn, err := database.NewSqlite()
	if err != nil {
		fmt.Errorf("error from initiate sqlite : %v ", err)
		panic(err)
	}

	//initiate command handler here
	cmdHandler := commandhandler.NewCommandHandler(sqliteConn)

	listen := listener.NewListener(cmdHandler)

	go func() {
		// listener on trigger start up
		listen.TriggerStartUp()
	}()
	//listener on trigger shutdown
	listen.ListenForShutdownEvent()

	newHandler := handler.NewHandler(cmdHandler)

	router := routers.NewRoutes(newHandler)
	appRoutes := router.V1(r)

	return appRoutes
}
