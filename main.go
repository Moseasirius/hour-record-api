package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func initDB() error {
	// 
	dsn := "root:mysqlpassword@tcp(127.0.0.1:3306)/hour_record?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai"
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(1 * time.Hour)
	return db.Ping()
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(w).Encode(map[string]any{
		"code": 0,
		"msg":  "pong, service ready",
	})
}

type UserReq struct {
	OpenId   string `json:"openid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}
func saveUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "method not allow"})
		return
	}
	var req UserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "参数错误"})
		return
	}
	_, err := db.Exec(`INSERT IGNORE INTO user(openid,nickname,avatar) VALUES (?,?,?)`, req.OpenId, req.Nickname, req.Avatar)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok"})
}

type StartRecordReq struct {
	UserId int64  `json:"user_id"`
	Remark string `json:"remark"`
}
func startRecordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "method not allow"})
		return
	}
	var req StartRecordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "参数错误"})
		return
	}
	var cnt int
	err := db.QueryRow(`SELECT count(1) FROM hour_record WHERE user_id=? AND end_time IS NULL`, req.UserId).Scan(&cnt)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": err.Error()})
		return
	}
	if cnt > 0 {
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "你有正在进行的工时，不能重复打卡"})
		return
	}
	_, err = db.Exec(`INSERT INTO hour_record(user_id,start_time,remark) VALUES (?,?,?)`, req.UserId, time.Now(), req.Remark)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "打卡成功"})
}

type EndRecordReq struct {
	UserId int64 `json:"user_id"`
}
func endRecordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "method not allow"})
		return
	}
	var req EndRecordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "参数错误"})
		return
	}
	res, err := db.Exec(`UPDATE hour_record SET end_time=? WHERE user_id=? AND end_time IS NULL`, time.Now(), req.UserId)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": err.Error()})
		return
	}
	affect, _ := res.RowsAffected()
	if affect == 0 {
		json.NewEncoder(w).Encode(map[string]any{"code": -1, "msg": "没有正在打卡的记录"})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "结束打卡成功"})
}

func main() {
	err := initDB()
	if err != nil {
		fmt.Println("数据库连接失败：", err)
		return
	}
	fmt.Println("db connect success")

	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/api/user/save", saveUserHandler)
	http.HandleFunc("/api/record/start", startRecordHandler)
	http.HandleFunc("/api/record/end", endRecordHandler)

	println("server running on 0.0.0.0:8080")
	err = http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		println("listen err:", err.Error())
	}
}
