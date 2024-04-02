package routers

import (
	"BruteHttp/Controllers"
	"context"
	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {

	ctx, cancel := context.WithCancel(context.Background())

	r := gin.Default()

	task := r.Group("/task")
	{

		task.GET("/start", func(c *gin.Context) {
			controller := Controllers.TaskController{Ctx: ctx}
			controller.StartTask(c)
		})
		task.GET("/cancel", func(c *gin.Context) {
			controller := Controllers.TaskController{Ctx: ctx, Cancel: cancel}
			controller.CancelTask(c)
		})
	}

	result := r.Group("/result")
	{
		result.POST("/ip")
		result.POST("/dns")
		result.POST("/http")
	}

	return r
}
