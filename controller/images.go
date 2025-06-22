package controller

import "github.com/gin-gonic/gin"

func (ctrl *Controller) GetImage(c *gin.Context) string {
	return
}

func (ctrl *Controller) UploadImage(c *gin.Context) string {
	c.JSON(200, gin.H{"message": "UploadImage endpoint is not implemented yet"})
	return "UploadImage endpoint is not implemented yet"
}
