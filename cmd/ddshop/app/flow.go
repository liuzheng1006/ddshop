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

package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zc2638/ddshop/core"
	"golang.org/x/sync/errgroup"
)

const (
	_payOrderParaNum = 2
)

var (
	successCh      = make(chan struct{}, 1)
	errCh          = make(chan error, 1)
	onceCart       = sync.Once{}
	onceCheckOrder = sync.Once{}
)

// flow 主流程
func flow(session *core.Session) error {
	logrus.Info("获取购物车")
	if err := session.GetCart(); err != nil {
		return err
	}
	if len(session.Cart.ProdList) == 0 {
		return core.ErrorNoValidProduct
	}
	onceCart.Do(func() {
		logrus.Info("-----------购物车守护程序启动--------------")
		core.WrapFun(session.GetCart)
	})
	logrus.Info("全选购物车")
	if err := session.CartAllCheck(); err != nil {
		return fmt.Errorf("全选购车车商品失败: %v", err)
	}

	logrus.Info("运力检查")
	_ = session.OrderFlashSale()

	logrus.Info("订单检查")
	if err := session.CheckOrder(); err != nil {
		return fmt.Errorf("检查订单失败: %v", err)
	}
	onceCheckOrder.Do(func() {
		logrus.Info("-----------检查订单守护程序启动--------------")
		core.WrapFun(session.CheckOrder)
	})

	logrus.Info("获取可预约时间")
	multiReserveTime, err := session.GetMultiReserveTime()
	if err != nil {
		return fmt.Errorf("获取可预约时间失败: %v", err)
	}
	if len(multiReserveTime) == 0 {
		return core.ErrorNoReserveTime
	}

	wg, _ := errgroup.WithContext(context.Background())
	for i := 0; i < _payOrderParaNum; i++ {
		for _, reserveTime := range multiReserveTime {
			sess := session.Clone()
			sess.UpdatePackageOrder(reserveTime)
			wg.Go(func() error {
				timeRange := session.GetReservedTimeRange()
				if err := sess.CreateOrder(context.Background()); err != nil {
					logrus.Warningf("提交订单(%s)失败: %v", timeRange, err)
					return err
				}
				logrus.Warningf("提交订单(%s)成功！", timeRange)
				successCh <- struct{}{}
				core.StopDaemonThread = true
				return nil
			})
		}
	}
	_ = wg.Wait()
	return nil
}
