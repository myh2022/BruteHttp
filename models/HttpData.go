package models

import (
	"BruteHttp/dao"
)

type HttpData struct {
	Id             int
	Site           string
	StatusCode     int
	Header         string
	Fingers        string
	IconPath       string
	ScreenshotPath string
	Time           string
	TaskName       string
}

func (HttpData) TableName() string {
	return "HttpData"
}

func (HttpData) InsertOrUpdateHttpData(httpDataList []HttpData) error {

	for _, httpData := range httpDataList {
		result := dao.Db.Where("site = ?", httpData.Site).First(&httpData)

		if result.RecordNotFound() {
			err := dao.Db.Create(&httpData).Error

			if err != nil {
				return err
			}
		} else {
			err := dao.Db.Model(&HttpData{}).Where("site = ?", httpData.Site).Updates(httpData).Error

			if err != nil {
				return err
			}
		}
	}

	return nil
}
