package controllers

/*
支付服务设计:
1. 一个账号在游戏冲的钱,可以在游戏所有区使用. 只需要(账号,钻石)对应关系
2. 一共有5个http请求,对每个http请求,查看外网的response是什么，结合代码,将不必要的字段删掉,只保留必要的回复字段
（目前只实现2个API：查询钻石余额，扣钱）
3. 内网保留一个加钻石的接口,点击钻石会加.注意首充.

一些问题
1. 如何在内网也发起请求?依靠 tencent_version ? 依靠 all_super_gm?
修改 TxpayMgr::buildReqCookie(),无论内外网都发请求
LoginCmdHandle::client_login()中 user_type 是客户端发给 FLServer的


具体API:
查询钻石余额
GET /v3/r/mpay/get_balance_m
参数(现有机制):
{
    "openid":"xxx账号名"
}
返回:
{
    "ret":0,
    "balance":66,
    "gen_balance":6,
}


扣除钻石:
GET /v3/r/mpay/pay_m
参数(现有机制):
{
	"openid":"xxx账号名",
    "amt":"扣除钻石的数量",
    "billno":订单号,
}

返回:
{
    "ret":0,
    "balance":钻石余额
    "billno":请求的订单号
}

充钻石:
GET /v3/r/mpay/rmb
参数(现有机制):
{
	"openid":"xxx账号名",
    "amt":"要冲的钻石的数量，只能为60,300,680,980,1280,1980,3280,6480",
    "billno":订单号
}

返回:
{
    "ret":0,
    "balance":当前钻石余额
}

初始时候,检查 table 不存在的话，创建一个 table,并创建 unique index on openid.

create table if not exists payinfo(
    openid varchar(128) not null primary key,
    stone_num bigint not null,
    gen_balance bigint not null,
    first_save int not null,
    save_amt bigint not null,
    save_sum bigint not null,
    cost_sum bigint not null,
    present_sum bigint not null,
)
function get_balance_m(openid){
    if openid 不存在,构造全0的json串返回
    if openid 存在,执行select语句.将查到结果返回:
}

function pay_m(openid, amt, billno){
    if openid 不存在,返回错误码
    if openid 存在,执行 update set stone_num,gen_balance,cost_sum 语句,返回对应json
}

function rmb_m(openid,amt, billno){
    if openid 不存在,往表格里加一条记录,bool isFirst = true;
    bool
update set stone_num
}
*/

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"payserver/models"
	u "payserver/utils"
	"strconv"
	"strings"
)

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}

	return b
}

// 解析 balance 字符串
func parseBalance(balance string) map[int]uint64 {
	balanceMap := make(map[int]uint64)
	eachZoneList := strings.Split(balance, ":")
	for _, element := range eachZoneList {
		zoneValue := strings.Split(element, "-")
		if len(zoneValue) != 2 {
			log.Printf("split balance %s,length!=2", element)
			continue
		}

		zoneid, err := strconv.Atoi(zoneValue[0])
		if err != nil {
			log.Printf("split balance convert zoneid %s failed", zoneValue[0])
			continue
		}
		zoneBalance, err := strconv.ParseUint(zoneValue[1], 10, 64)
		if err != nil {
			log.Printf("split balance convert balance %s failed", zoneValue[1])
			continue
		}
		balanceMap[zoneid] = zoneBalance
	}
	return balanceMap
}

// 将所有区的 balanceStr 合在一起
func saveBalance(bm map[int]uint64) string {
	balanceStr := ""
	blanceArray := []string{}
	for key, value := range bm {
		s := fmt.Sprintf("%d-%d", key, value)
		blanceArray = append(blanceArray, s)
	}

	if len(blanceArray) > 0 {
		balanceStr = strings.Join(blanceArray, ":")
	}
	return balanceStr
}

func getZoneid(zonestr string) int {
	zoneid := 0
	tmpArray := strings.Split(zonestr, "_")
	if len(tmpArray) != 2 {
		return zoneid
	}

	id, err := strconv.Atoi(tmpArray[0])
	if err != nil {
		return zoneid
	}
	return id
}

// 请求参数(目前只用 openid 字段):
// appid=1450015851&openid=4001&openkey=&pf=&pfkey=&sig=3qpf4g1Jy/ObW+68O+5tPbbLJFA=&ts=1560193277&zoneid=9022_100460844322950
var GetBalanceM = func(w http.ResponseWriter, r *http.Request) {
	log.Printf("GetBalanceM %v %v %v", r.Method, r.URL, r.Proto)
	openid := r.URL.Query().Get("openid")

	if openid == "" {
		u.Respond(w, map[string]interface{}{"ret": 9999})
		return
	}
	zoneid := getZoneid(r.URL.Query().Get("zoneid"))
	if zoneid == 0 {
		u.Respond(w, map[string]interface{}{"ret": 8888, "err": "Please input zoneinfo"})
		return
	}

	zoneOpenid := fmt.Sprintf("%d_%s", zoneid, openid)
	data := models.GetOpenidPayInfo(zoneOpenid)
	if data == nil {
		tmppayinfo := &models.PayInfo{}
		json.NewEncoder(w).Encode(tmppayinfo)
		log.Printf("Could not find openid=%s,return default", zoneOpenid)
	} else {
		json.NewEncoder(w).Encode(data)
		log.Printf("%#v", data)
	}
}

// 请求参数(目前只用 openid,amt,billno):
// amt=90&appid=1450015851&billno=1869376062307901441&openid=4001&openkey=&pf=&pfkey=&sig=M5H2lLV48lV7gbmQG3YAOuC4LZI=&ts=1560193307&zoneid=9022_100460844322950
var PayM = func(w http.ResponseWriter, r *http.Request) {
	log.Printf("PayM %v %v %v", r.Method, r.URL, r.Proto)
	openid := r.URL.Query().Get("openid")
	if openid == "" {
		u.Respond(w, map[string]interface{}{"ret": 9999})
		return
	}

	zoneid := getZoneid(r.URL.Query().Get("zoneid"))
	if zoneid == 0 {
		u.Respond(w, map[string]interface{}{"ret": 6666, "err": "Please input zoneinfo"})
		return
	}

	amt, err := strconv.Atoi(r.URL.Query().Get("amt"))
	if err != nil {
		u.Respond(w, map[string]interface{}{"ret": 8888})
		return
	}

	uint64amt := uint64(amt)
	zoneOpenid := fmt.Sprintf("%d_%s", zoneid, openid)
	data := models.GetOpenidPayInfo(zoneOpenid)
	// 玩家的openid不存在
	if data == nil {
		u.Respond(w, map[string]interface{}{"ret": 7777})
		return
	}

	// 余额不足
	if data.Balance < uint64amt {
		u.Respond(w, map[string]interface{}{"ret": 7777, "balance": -1})
		return
	}

	retJson := map[string]interface{}{"ret": 0}
	billno := r.URL.Query().Get("billno")
	retJson["billno"] = billno
	// 优先消耗掉赠送的钻石
	if data.GenBalance > 0 {
		tmpCostAmt := min(uint64amt, data.GenBalance)
		uint64amt -= tmpCostAmt
		data.CostSum += tmpCostAmt
		data.Balance -= tmpCostAmt
		retJson["used_gen_amt"] = tmpCostAmt
		data.GenBalance -= tmpCostAmt
	}

	if uint64amt > 0 {
		data.Balance -= uint64amt
		data.CostSum += uint64amt
	}
	models.SaveToDB(data)

	retJson["balance"] = data.Balance
	json.NewEncoder(w).Encode(retJson)
	log.Printf("%#v", data)
}

// 请求参数(openid,amt,billno)
// amt=90&billno=1869376062307901441&openid=4001
var RmbM = func(w http.ResponseWriter, r *http.Request) {
	log.Printf("RmbM %v %v %v", r.Method, r.URL, r.Proto)
	// key:要冲的钻石的数量，只能为60,300,680,980,1280,1980,3280,6480"
	// value:赠送的钻石数量
	validAmounts := map[int]uint64{
		60:   6,
		300:  33,
		680:  78,
		980:  118,
		1280: 168,
		1980: 268,
		3280: 498,
		6480: 1288,
	}

	openid := r.URL.Query().Get("openid")
	if openid == "" {
		u.Respond(w, map[string]interface{}{"ret": 9999})
		return
	}

	zoneid := getZoneid(r.URL.Query().Get("zoneid"))
	if zoneid == 0 {
		u.Respond(w, map[string]interface{}{"ret": 6666, "err": "Please input zoneinfo"})
		return
	}

	amt, err := strconv.Atoi(r.URL.Query().Get("amt"))
	if err != nil {
		u.Respond(w, map[string]interface{}{"ret": 8888})
		return
	}

	uint64amt := uint64(amt)

	genNum, ok := validAmounts[amt]
	if !ok {
		genNum = 0
	}

	billno := r.URL.Query().Get("billno")
	zoneOpenid := fmt.Sprintf("%d_%s", zoneid, openid)
	data := models.GetOpenidPayInfo(zoneOpenid)
	if data == nil {
		data = &models.PayInfo{}
		data.Openid = zoneOpenid
		data.Balance = uint64amt + genNum
		data.GenBalance = genNum
		data.SaveAmt = uint64amt
		data.SaveSum = uint64amt + genNum
		data.PresentSum = genNum
		data.Billno = billno
		w.Header().Add("Content-Type", "application/json")
		models.CreateToDB(data)
		json.NewEncoder(w).Encode(data)
	} else {
		data.Balance += (uint64amt + genNum)
		data.GenBalance += genNum
		data.FirstSave = 0
		data.SaveAmt += (uint64amt)
		data.SaveSum += (uint64amt + genNum)
		data.PresentSum += genNum
		data.Billno = billno
		models.SaveToDB(data)
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
	log.Printf("%#v", data)
}

var DefaultReturn = func(w http.ResponseWriter, r *http.Request) {
	u.Respond(w, map[string]interface{}{"ret": 9999})
	return
}

// 请求执行批处理
// 参数:id=1
var Bgm = func(w http.ResponseWriter, r *http.Request) {
	bgmid, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		u.Respond(w, map[string]interface{}{"ret": 8888})
		return
	}

	resp, err := http.Get("http://192.168.12.136:8090/pages/viewpage.action?pageId=327905")
	if err != nil {
		u.Respond(w, map[string]interface{}{"ret": 7777, "msg": "webpage http://192.168.12.136:8090/pages/viewpage.action?pageId=327905 not available"})
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		u.Respond(w, map[string]interface{}{"ret": 6666, "msg": "read msg failed"})
		return
	}

}
