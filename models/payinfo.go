package models

import (
	u "payserver/utils"
	"github.com/jinzhu/gorm"	
)

type PayInfo struct {
	gorm.Model
	Openid string `gorm:"PRIMARY_KEY;NOT NULL" json:"openid"`	
	GenBalance uint64 `gorm:"NOT NULL" json:"gen_balance"`
	FirstSave int `gorm:"NOT NULL;default:1" json:"first_save"`
	SaveAmt uint64 `gorm:"NOT NULL" json:"save_amt"`
	SaveSum uint64 `gorm:"NOT NULL" json:"save_sum"`
	CostSum uint64 `gorm:"NOT NULL" json:"cost_sum"`
	PresentSum uint64 `gorm:"NOT NULL" json:"present_sum"`
	Ret int `gorm:"NOT NULL" json:"ret"`
	Balance uint64 `gorm:"NOT NULL" json:"balance"`
	Billno string `json:"billno"`	
}

func GetOpenidPayInfo(openid string) (*PayInfo) {
	payinfo := &PayInfo{}	
	err := GetDB().Table("pay_infos").Where("openid = ?", openid).First(payinfo).Error
	if err != nil{
		return nil
	}
	return payinfo
}

func CreateToDB(info *PayInfo) {
	GetDB().Create(info)
}

func SaveToDB(info *PayInfo){
	GetDB().Save(info)
}

/*
 This struct function validate the required parameters sent through the http request body

returns message and true if the requirement is met
*/
func (payinfo *PayInfo) Validate() (map[string] interface{}, bool) {
	//All the required parameters are present
	return u.Message(true, "success"), true
}

func (payinfo *PayInfo) Create() (map[string] interface{}) {
	resp := u.Message(true, "success")
	resp["contact"] = payinfo
	return resp
}