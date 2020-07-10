package main

import (
	"github.com/gorilla/mux"	
	"os"
	"fmt"
	"net/http"
	"payserver/controllers"
)

func main() {

	router := mux.NewRouter()

	// 新增一个用户
	// get该用户的钻石,对应 get_balance_m
	// set该用户的钻石
	// add该用户的钻石
	// sub该用户的钻石,pay_m
	// 取消支付接口,cancel_pay_m
	// 直购接口,buy_goods_m
	// 赠送接口,present_m	
	router.HandleFunc("/v3/r/mpay/get_balance_m", controllers.GetBalanceM).Methods("GET")
	router.HandleFunc("/v3/r/mpay/rmb_m", controllers.RmbM).Methods("GET")
	router.HandleFunc("/v3/r/mpay/pay_m", controllers.PayM).Methods("GET")
	router.HandleFunc("/v3/r/mpay/cancel_pay_m", controllers.DefaultReturn).Methods("GET")
	router.HandleFunc("/v3/r/mpay/present_m", controllers.DefaultReturn).Methods("GET")
	router.HandleFunc("/v3/r/mpay/buy_goods_m", controllers.DefaultReturn).Methods("GET")
	router.HandleFunc("/v3/r/mpay/bgm", controllers.Bgm).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000" //localhost
	}

	fmt.Println(port)

	err := http.ListenAndServe(":" + port, router) //Launch the app, visit localhost:8000/api
	if err != nil {
		fmt.Print(err)
	}
}
