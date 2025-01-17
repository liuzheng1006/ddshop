// Copyright © 2022 zc2638 <zc2638@qq.com>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

func NewSession(cookie string, interval int64) *Session {
	if !strings.HasPrefix(cookie, "DDXQSESSID=") {
		cookie = "DDXQSESSID=" + cookie
	}

	header := make(http.Header)
	header.Set("Host", "maicai.api.ddxq.mobi")
	header.Set("user-agent", "Mozilla/5.0 (Linux; Android 9; LIO-AN00 Build/LIO-AN00; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/92.0.4515.131 Mobile Safari/537.36 xzone/9.47.0 station_id/null")
	header.Set("accept", "application/json, text/plain, */*")
	header.Set("content-type", "application/x-www-form-urlencoded")
	header.Set("origin", "https://wx.m.ddxq.mobi")
	header.Set("x-requested-with", "com.yaya.zone")
	header.Set("sec-fetch-site", "same-site")
	header.Set("sec-fetch-mode", "cors")
	header.Set("sec-fetch-dest", "empty")
	header.Set("referer", "https://wx.m.ddxq.mobi/")
	header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	header.Set("cookie", cookie)

	client := resty.New()
	client.Header = header
	return &Session{
		client:   client,
		Interval: interval,

		apiVersion:   "9.50.0",
		appVersion:   "2.83.0",
		channel:      "applet",
		appClientID:  "4",
		Cart:         &Cart{},
		Order:        &Order{},
		PackageOrder: &PackageOrder{},
	}
}

type Session struct {
	client   *resty.Client
	Interval int64 // 间隔请求时间(ms)

	channel     string
	apiVersion  string
	appVersion  string
	appClientID string

	UserID   string
	Address  *AddressItem
	BarkId   string
	PayType  int
	CartMode int

	Cart         *Cart
	Order        *Order
	PackageOrder *PackageOrder
}

func (s *Session) Clone() *Session {
	return &Session{
		client:   s.client,
		Interval: s.Interval,

		UserID:   s.UserID,
		Address:  s.Address,
		BarkId:   s.BarkId,
		PayType:  s.PayType,
		CartMode: s.CartMode,

		Cart:         s.Cart,
		Order:        s.Order,
		PackageOrder: s.PackageOrder,
		apiVersion:   s.apiVersion,
		appVersion:   s.appVersion,
		channel:      s.channel,
		appClientID:  s.appClientID,
	}
}

func (s *Session) execute(ctx context.Context, request *resty.Request, method, url string) (*resty.Response, error) {
	urlAbbr := strings.Split(url, "/")
	actionName := urlAbbr[len(urlAbbr)-2] + "/" + strings.Split(urlAbbr[len(urlAbbr)-1], "?")[0]
	if actionName == "order/addNewOrder" {
		logrus.Infof("提交订单中, 预约时间段(%s),下单金额(%s)", s.GetReservedTimeRange(), s.Order.Price)
	}
	if ctx != nil {
		request.SetContext(ctx)
	}
	resp, err := request.Execute(method, url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	// 当用户访问有可能对网站造成安全威胁的URL时，会收到405报错，提示访问被WAF拦截。
	if resp.StatusCode() == http.StatusMethodNotAllowed {
		return nil, ErrMethodNotAllowed
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("statusCode: %d, body: %s", resp.StatusCode(), resp.String())
	}

	result := gjson.ParseBytes(resp.Body())
	code := result.Get("code").Num
	switch code {
	case 0:
		return resp, nil
	case 1, -3000, -3001, -3100:
		// -3000:检查订单报错，当前人多拥挤，请稍后尝试刷新页面
		// -3100:检查订单失败，加载失败，请重新尝试
		// -3001:提交订单报错拥挤
	case 5003:
		return nil, ErrNoValidFreight
	case -1:
		return nil, ErrOperator
	default:
		return nil, fmt.Errorf("无法识别的状态码: %v", resp.String())
	}
	duration := time.Duration(s.Interval + rand.Int63n(s.Interval/2))
	//logrus.Warningf("将在 %dms 后重试, 当前人多拥挤(%v)(%s)", duration, actionName, resp.String())
	time.Sleep(duration * time.Millisecond)
	return s.execute(nil, request, method, url)
}
func (s *Session) GetReservedTimeRange() string {
	startTime := time.Unix(int64(s.PackageOrder.PaymentOrder.ReservedTimeStart), 0).Format("2006/01/02 15:04:05")
	endTime := time.Unix(int64(s.PackageOrder.PaymentOrder.ReservedTimeEnd), 0).Format("2006/01/02 15:04:05")
	return startTime + "——" + endTime
}

func (s *Session) buildHeader() http.Header {
	header := make(http.Header)
	header.Set("ddmc-city-number", s.Address.CityNumber)
	header.Set("ddmc-os-version", "undefined")
	header.Set("ddmc-channel", s.channel)
	header.Set("ddmc-api-version", s.apiVersion)
	header.Set("ddmc-build-version", s.appVersion)
	header.Set("ddmc-app-client-id", s.appClientID)
	header.Set("ddmc-ip", "")
	header.Set("ddmc-station-id", s.Address.StationId)
	header.Set("ddmc-uid", s.UserID)
	if len(s.Address.Location.Location) == 2 {
		header.Set("ddmc-longitude", strconv.FormatFloat(s.Address.Location.Location[0], 'f', -1, 64))
		header.Set("ddmc-latitude", strconv.FormatFloat(s.Address.Location.Location[1], 'f', -1, 64))
	}
	return header
}

func (s *Session) buildURLParams(needAddress bool) url.Values {
	params := url.Values{}
	params.Add("channel", s.channel)
	params.Add("api_version", s.apiVersion)
	params.Add("app_version", s.appVersion)
	params.Add("app_client_id", s.appClientID)
	params.Add("applet_source", "")
	params.Add("h5_source", "")
	params.Add("sharer_uid", "")
	params.Add("s_id", "")
	params.Add("openid", "")

	params.Add("uid", s.UserID)
	if needAddress {
		params.Add("address_id", s.Address.Id)
		params.Add("station_id", s.Address.StationId)
		params.Add("city_number", s.Address.CityNumber)
		if len(s.Address.Location.Location) == 2 {
			params.Add("longitude", strconv.FormatFloat(s.Address.Location.Location[0], 'f', -1, 64))
			params.Add("latitude", strconv.FormatFloat(s.Address.Location.Location[1], 'f', -1, 64))
		}
	}

	// TODO 不知道是不是必须
	params.Add("device_token", "")

	// TODO 计算方式?
	params.Add("nars", "")
	params.Add("sesi", "")
	return params
}

func (s *Session) Choose() error {
	if err := s.chooseAddr(); err != nil {
		return err
	}
	if err := s.choosePay(); err != nil {
		return err
	}
	if err := s.chooseCartMode(); err != nil {
		return err
	}
	return nil
}

func (s *Session) chooseAddr() error {
	addrMap, err := s.GetAddress()
	if err != nil {
		return fmt.Errorf("获取收货地址失败: %v", err)
	}
	addrs := make([]string, 0, len(addrMap))
	for k := range addrMap {
		addrs = append(addrs, k)
	}

	//var addr string
	//sv := &survey.Select{
	//	Message: "请选择收货地址",
	//	Options: addrs,
	//}
	//if err := survey.AskOne(sv, &addr); err != nil {
	//	return fmt.Errorf("选择收货地址错误: %v", err)
	//}

	address, ok := addrMap[addrs[0]]
	if !ok {
		return errors.New("请选择正确的收货地址")
	}
	s.Address = &address
	logrus.Infof("已选择收货地址: %s %s", s.Address.Location.Address, s.Address.AddrDetail)
	return nil
}

const (
	paymentAlipay = "支付宝"
	paymentWechat = "微信"
)

func (s *Session) choosePay() error {
	//var payName string
	//sv := &survey.Select{
	//	Message: "请选择支付方式",
	//	Options: []string{paymentWechat, paymentAlipay},
	//	Default: paymentWechat,
	//}
	//if err := survey.AskOne(sv, &payName); err != nil {
	//	return fmt.Errorf("选择支付方式错误: %v", err)
	//}
	//
	//switch payName {
	//case paymentAlipay:
	//	s.PayType = 2
	//case paymentWechat:
	//	s.PayType = 4
	//default:
	//	return fmt.Errorf("无法识别的支付方式: %s", payName)
	//}
	s.PayType = 4
	return nil
}

const (
	cartModeAvailable = "结算所有有效商品(不包括换购)"
	cartModeAll       = "结算所有勾选商品(包括换购)"
)

func (s *Session) chooseCartMode() error {
	//var cartDesc string
	//sv := &survey.Select{
	//	Message: "请选择购物车商品结算模式",
	//	Options: []string{cartModeAvailable, cartModeAll},
	//	Default: cartModeAvailable,
	//}
	//if err := survey.AskOne(sv, &cartDesc); err != nil {
	//	return fmt.Errorf("选择购物车商品结算模式错误: %v", err)
	//}
	//
	//switch cartDesc {
	//case cartModeAvailable:
	//	s.CartMode = 1
	//case cartModeAll:
	//	s.CartMode = 2
	//default:
	//	return fmt.Errorf("无法识别的购物车商品结算模式: %s", cartDesc)
	//}
	s.CartMode = 2
	return nil
}
