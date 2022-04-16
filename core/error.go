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

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrorNoValidProduct     = Error("无有效商品")
	ErrorNoReserveTime      = Error("无可预约时间段")
	ErrNoValidFreight       = Error("运费支付金额不正确")
	ErrOperator             = Error("操作失败")
	ErrMethodNotAllowed     = Error("MethodNotAllowed")
	ErrorNoValidReserveTime = Error("预定时间小于当前时间，不合法")
	ErrCapacityFull         = Error("由于近期疫情问题，配送运力紧张，本站点当前运力已约满")
)
