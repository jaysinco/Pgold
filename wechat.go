package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"
)

func procWechat(w http.ResponseWriter, r *http.Request, mlog StdLogger, db *sql.DB) {
	signature := mkSignature("jaysinco", r.Form.Get("timestamp"), r.Form.Get("nonce"))
	if signature == r.Form.Get("signature") {
		mlog("receive request coming from wechat public platform!")
		if r.Method == "GET" && r.Form.Get("echostr") != "" {
			mlog("send echostr for wechat public platform server configuration")
			fmt.Fprintf(w, r.Form.Get("echostr"))
			return
		}
	} else {
		mlog("receive request not coming from wechat public platform, ignore!")
		return
	}

	rbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		mlog("[ERROR] read request body: %v", err)
		return
	}
	mlog("[POST] %s", strings.Replace(string(rbody), "\n", "", -1))
	recMsg := new(WxRecvMsg)
	if err := xml.Unmarshal(rbody, recMsg); err != nil {
		mlog("[ERROR] parse xml: %v", err)
		return
	}
	rplStr := genReplyStr(recMsg)
	mlog("[SEND] %s", rplStr)
	if _, err := fmt.Fprintf(w, rplStr); err != nil {
		mlog("[ERROR] write response: %v", err)
	}
}

func genReplyStr(recMsg *WxRecvMsg) string {
	reply := "success"
	if recMsg.Content == "who" {
		reply = mkWxRplStr(recMsg, "Jaysinco")
	}
	return reply
}

func mkWxRplStr(m *WxRecvMsg, c string) string {
	s := new(WxRplMsg)
	s.FromUserName = str2CDATA(m.ToUserName)
	s.ToUserName = str2CDATA(m.FromUserName)
	s.CreateTime = time.Duration(time.Now().Unix())
	s.MsgType = str2CDATA("text")
	s.Content = str2CDATA(c)
	b, _ := xml.Marshal(s)
	return string(b)
}

func mkSignature(token, timeStamp, nonce string) string {
	rst := []string{token, timeStamp, nonce}
	sort.Strings(rst)
	sig := sha1.New()
	io.WriteString(sig, strings.Join(rst, ""))
	return fmt.Sprintf("%x", sig.Sum(nil))
}

type WxRecvMsg struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string
	FromUserName string
	CreateTime   time.Duration
	MsgType      string
	Content      string
	MsgId        string
}

type WxRplMsg struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   CDATAText
	FromUserName CDATAText
	CreateTime   time.Duration
	MsgType      CDATAText
	Content      CDATAText
}

type CDATAText struct {
	Text string `xml:",innerxml"`
}

func str2CDATA(s string) CDATAText {
	return CDATAText{Text: fmt.Sprintf("<![CDATA[%s]]>", s)}
}
